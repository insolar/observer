//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package api

import (
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/application/appfoundation"

	"github.com/insolar/observer/internal/app/observer/postgres"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"

	"github.com/labstack/echo/v4"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/internal/models"

	"github.com/sirupsen/logrus"
)

type Clock interface {
	Now() time.Time
}

type DefaultClock struct{}

func (c *DefaultClock) Now() time.Time {
	return time.Now()
}

type ObserverServer struct {
	db    *pg.DB
	log   *logrus.Logger
	clock Clock
	fee   *big.Int
	price string
}

func NewObserverServer(db *pg.DB, log *logrus.Logger, fee *big.Int, clock Clock, price string) *ObserverServer {
	return &ObserverServer{db: db, log: log, clock: clock, fee: fee, price: price}
}

func (s *ObserverServer) GetMigrationAddresses(ctx echo.Context, params GetMigrationAddressesParams) error {
	limit := params.Limit
	if limit <= 0 || limit > 1000 {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("`limit` should be in range [1, 1000]"))
	}

	query := s.db.Model(&models.MigrationAddress{}).
		Where("wasted = false")
	if params.Index != nil {
		id, err := strconv.ParseInt(*params.Index, 10, 64)
		if err != nil {
			s.log.Error(err)
			return ctx.JSON(http.StatusBadRequest, struct{}{})
		}
		query = query.Where("id > ?", id)
	}
	var result []models.MigrationAddress
	err := query.Order("id").Limit(limit).Select(&result)
	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	resJSON := make(ResponsesAddressesYaml, len(result))
	for i := 0; i < len(result); i++ {
		resJSON[i].Address = result[i].Addr
		resJSON[i].Index = strconv.FormatInt(result[i].ID, 10)
	}
	return ctx.JSON(http.StatusOK, resJSON)
}

// GetMigrationAddressCount returns the total number of non-assigned migration addresses
func (s *ObserverServer) GetMigrationAddressCount(ctx echo.Context) error {
	count, err := s.db.Model(&models.MigrationAddress{}).
		Where("wasted = false").
		Count()
	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, ResponsesAddressCountYaml{
		Count: count,
	})
}

func (s *ObserverServer) TransactionsDetails(ctx echo.Context, txID string) error {
	panic("implement me")
}

// CloseTransactions returns a list of closed transactions (only with statuses `received` and `failed`).
func (s *ObserverServer) ClosedTransactions(ctx echo.Context, params ClosedTransactionsParams) error {
	limit := params.Limit
	if limit <= 0 || limit > 1000 {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("`limit` should be in range [1, 1000]"))
	}

	var (
		pulseNumber    int64
		sequenceNumber int64
		err            error
	)
	if params.Index != nil {
		pulseNumber, sequenceNumber, err = checkIndex(*params.Index)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, NewSingleMessageError(err.Error()))
		}
	}

	var result []models.Transaction
	query := s.db.Model(&models.Transaction{}).
		Where("status_finished = ?", true)
	query, err = component.OrderByIndex(query, params.Order, pulseNumber, sequenceNumber, params.Index != nil, models.TxIndexTypeFinishPulseRecord)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError(err.Error()))
	}
	err = query.
		Limit(limit).
		Select(&result)
	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	resJSON := make([]interface{}, len(result))
	for i := 0; i < len(result); i++ {
		resJSON[i] = TxToAPITx(result[i], models.TxIndexTypeFinishPulseRecord)
	}
	return ctx.JSON(http.StatusOK, resJSON)
}

func isInt(s string) bool {
	s = strings.TrimPrefix(s, "-")
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func (s *ObserverServer) Fee(ctx echo.Context, amount string) error {
	if !isInt(amount) {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("invalid amount"))
	}
	if strings.HasPrefix(amount, "-") {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("negative amount"))
	}

	return ctx.JSON(http.StatusOK, ResponsesFeeYaml{Fee: s.fee.String()})
}

func (s *ObserverServer) Member(ctx echo.Context, reference string) error {
	var migrationAddress string
	ref, errMsg := s.checkReference(reference)
	if errMsg != nil {
		if appfoundation.IsEthereumAddress(reference) {
			migrationAddress = reference
		} else {
			return ctx.JSON(http.StatusBadRequest, *errMsg)
		}
	}
	byMigrationAddress := migrationAddress != ""

	var member *models.Member
	var err error
	var memberReference []byte

	if byMigrationAddress {
		member, err = component.GetMemberByMigrationAddress(ctx.Request().Context(), s.db, migrationAddress)
		if member != nil {
			memberReference = member.Reference
		}
	} else {
		memberReference = ref.Bytes()
		member, err = component.GetMember(ctx.Request().Context(), s.db, memberReference)
	}
	if err != nil {
		if err == component.ErrReferenceNotFound {
			return ctx.NoContent(http.StatusNoContent)
		}
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	deposits, err := component.GetDeposits(ctx.Request().Context(), s.db, memberReference)
	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, MemberToAPIMember(*member, *deposits, s.clock.Now().Unix(), byMigrationAddress))
}

func (s *ObserverServer) Balance(ctx echo.Context, reference string) error {
	ref, errMsg := s.checkReference(reference)
	if errMsg != nil {
		return ctx.JSON(http.StatusBadRequest, *errMsg)
	}

	member, err := component.GetMemberBalance(ctx.Request().Context(), s.db, ref.Bytes())
	if err != nil {
		if err == component.ErrReferenceNotFound {
			return ctx.NoContent(http.StatusNoContent)
		}
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, ResponsesMemberBalanceYaml{Balance: member.Balance})
}

func (s *ObserverServer) MemberTransactions(ctx echo.Context, reference string, params MemberTransactionsParams) error {
	ref, errMsg := s.checkReference(reference)
	if errMsg != nil {
		return ctx.JSON(http.StatusBadRequest, *errMsg)
	}

	limit := params.Limit
	if limit <= 0 || limit > 1000 {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("`limit` should be in range [1, 1000]"))
	}

	var errorMsg ErrorMessage

	var txs []models.Transaction
	query := s.db.Model(&txs)

	query, err := component.FilterByMemberReferenceAndDirection(query, ref, params.Direction)
	if err != nil {
		errorMsg.Error = append(errorMsg.Error, err.Error())
	}

	query = s.getTransactions(query, &errorMsg, params.Status, params.Type, params.Index, params.Order)

	if len(errorMsg.Error) > 0 {
		return ctx.JSON(http.StatusBadRequest, errorMsg)
	}

	err = query.Limit(limit).Select()

	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	if len(txs) == 0 {
		return ctx.NoContent(http.StatusNoContent)
	}

	res := SchemasTransactions{}
	for _, t := range txs {
		res = append(res, TxToAPITx(t, models.TxIndexTypePulseRecord))
	}
	return ctx.JSON(http.StatusOK, res)
}

func (s *ObserverServer) Notification(ctx echo.Context) error {
	res, err := component.GetNotification(ctx.Request().Context(), s.db)
	if err != nil {
		if err == component.ErrNotificationNotFound {
			return ctx.NoContent(http.StatusNoContent)
		}
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, ResponsesNotificationInfoYaml{
		Notification: res.Message,
	})
}

func (s *ObserverServer) Transaction(ctx echo.Context, txIDStr string) error {
	txIDStr = strings.TrimSpace(txIDStr)

	if len(txIDStr) == 0 {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("empty tx_id"))
	}

	txIDStr, err := url.QueryUnescape(txIDStr)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("error unescaping tx_id parameter"))
	}

	txID, err := insolar.NewRecordReferenceFromString(txIDStr)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("tx_id wrong format"))
	}

	tx, err := component.GetTx(ctx.Request().Context(), s.db, txID.Bytes())
	if err != nil {
		if err == component.ErrTxNotFound {
			return ctx.NoContent(http.StatusNoContent)
		}
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, TxToAPITx(*tx, models.TxIndexTypePulseRecord))
}

func (s *ObserverServer) TransactionsSearch(ctx echo.Context, params TransactionsSearchParams) error {
	limit := params.Limit
	if limit <= 0 || limit > 1000 {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("`limit` should be in range [1, 1000]"))
	}

	var errorMsg ErrorMessage
	var err error

	var txs []models.Transaction
	query := s.db.Model(&txs)

	if params.Value != nil {
		query, err = component.FilterByValue(query, *params.Value)
		if err != nil {
			errorMsg.Error = append(errorMsg.Error, err.Error())
		}
	}

	query = s.getTransactions(query, &errorMsg, params.Status, params.Type, params.Index, params.Order)
	if len(errorMsg.Error) > 0 {
		return ctx.JSON(http.StatusBadRequest, errorMsg)
	}
	err = query.Limit(params.Limit).Select()

	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	if len(txs) == 0 {
		return ctx.NoContent(http.StatusNoContent)
	}

	res := SchemasTransactions{}
	for _, t := range txs {
		res = append(res, TxToAPITx(t, models.TxIndexTypePulseRecord))
	}
	return ctx.JSON(http.StatusOK, res)
}

func (s *ObserverServer) SupplyStats(ctx echo.Context) error {
	repo := postgres.NewSupplyStatsRepository(s.db)
	xr := component.NewStatsManager(s.log, repo)
	result, err := xr.Supply()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, "")
	}

	return ctx.JSON(http.StatusOK, ResponsesSupplyStatsYaml{
		TotalSupply:       result.Total(),
		MaxSupply:         result.Max(),
		CirculatingSupply: result.Circulating(),
	})
}

func (s *ObserverServer) SupplyStatsCirculating(ctx echo.Context) error {
	repo := postgres.NewSupplyStatsRepository(s.db)
	xr := component.NewStatsManager(s.log, repo)
	result, err := xr.Circulating()
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "")
	}

	return ctx.String(http.StatusOK, result)
}

func (s *ObserverServer) SupplyStatsMax(ctx echo.Context) error {
	repo := postgres.NewSupplyStatsRepository(s.db)
	xr := component.NewStatsManager(s.log, repo)
	result, err := xr.Max()
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "")
	}

	return ctx.String(http.StatusOK, result)
}

func (s *ObserverServer) SupplyStatsTotal(ctx echo.Context) error {
	repo := postgres.NewSupplyStatsRepository(s.db)
	xr := component.NewStatsManager(s.log, repo)
	result, err := xr.Total()
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "")
	}

	return ctx.String(http.StatusOK, result)
}

func (s *ObserverServer) MarketStats(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, ResponsesMarketStatsYaml{
		Price: s.price,
	})
}

func (s *ObserverServer) NetworkStats(ctx echo.Context) error {
	repo := postgres.NewNetworkStatsRepository(s.db)
	result, err := repo.LastStats()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, "")
	}

	return ctx.JSON(http.StatusOK, ResponsesNetworkStatsYaml{
		Accounts:              result.TotalAccounts,
		CurrentTPS:            result.CurrentTPS,
		LastMonthTransactions: result.MonthTransactions,
		MaxTPS:                result.MaxTPS,
		Nodes:                 result.Nodes,
		TotalTransactions:     result.TotalTransactions,
	})
}

func checkIndex(i string) (int64, int64, error) {
	index := strings.Split(i, ":")
	if len(index) != 2 {
		return 0, 0, errors.New("Query parameter 'index' should have the '<pulse_number>:<sequence_number>' format.") // nolint
	}
	var err error
	var pulseNumber, sequenceNumber int64
	pulseNumber, err = strconv.ParseInt(index[0], 10, 64)
	if err != nil {
		return 0, 0, errors.New("Query parameter 'index' should have the '<pulse_number>:<sequence_number>' format.") // nolint
	}
	sequenceNumber, err = strconv.ParseInt(index[1], 10, 64)
	if err != nil {
		return 0, 0, errors.New("Query parameter 'index' should have the '<pulse_number>:<sequence_number>' format.") // nolint
	}
	return pulseNumber, sequenceNumber, nil
}

func (s *ObserverServer) getTransactions(
	query *orm.Query, errorMsg *ErrorMessage, status, typeParam, index, order *string,
) *orm.Query {
	var err error
	if status != nil {
		query, err = component.FilterByStatus(query, *status)
		if err != nil {
			errorMsg.Error = append(errorMsg.Error, err.Error())
		}
	}

	if typeParam != nil {
		query, err = component.FilterByType(query, *typeParam)
		if err != nil {
			errorMsg.Error = append(errorMsg.Error, err.Error())
		}
	}

	var pulseNumber int64
	var sequenceNumber int64
	byIndex := false
	if index != nil {
		pulseNumber, sequenceNumber, err = checkIndex(*index)
		if err != nil {
			errorMsg.Error = append(errorMsg.Error, err.Error())
		} else {
			byIndex = true
		}
	}

	query, err = component.OrderByIndex(query, order, pulseNumber, sequenceNumber, byIndex, models.TxIndexTypePulseRecord)
	if err != nil {
		errorMsg.Error = append(errorMsg.Error, err.Error())
	}
	return query
}

func (s *ObserverServer) checkReference(referenceRow string) (*insolar.Reference, *ErrorMessage) {
	referenceRow = strings.TrimSpace(referenceRow)
	var errMsg ErrorMessage

	if len(referenceRow) == 0 {
		errMsg = NewSingleMessageError("empty reference")
		return nil, &errMsg
	}

	reference, err := url.QueryUnescape(referenceRow)
	if err != nil {
		errMsg = NewSingleMessageError("error unescaping reference parameter")
		return nil, &errMsg
	}

	ref, err := insolar.NewReferenceFromString(reference)
	if err != nil {
		errMsg = NewSingleMessageError("reference wrong format")
		return nil, &errMsg
	}
	return ref, nil
}
