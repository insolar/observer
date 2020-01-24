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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/application/appfoundation"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	apiconfiguration "github.com/insolar/observer/configuration/api"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
)

type ObserverServer struct {
	db       *pg.DB
	log      insolar.Logger
	pStorage observer.PulseStorage
	config   apiconfiguration.Configuration
}

func NewObserverServer(db *pg.DB, log insolar.Logger, pStorage observer.PulseStorage, config apiconfiguration.Configuration) *ObserverServer {
	return &ObserverServer{db: db, log: log, pStorage: pStorage, config: config}
}

func (s *ObserverServer) IsMigrationAddress(ctx echo.Context, ethereumAddress string) error {
	s.setExpire(ctx, 1*time.Minute)

	count, err := s.db.Model(&models.MigrationAddress{}).
		Where("addr = ?", ethereumAddress).
		Count()

	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, ResponsesIsMigrationAddressYaml{
		IsMigrationAddress: count > 0,
	})
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
	s.setExpire(ctx, 10*time.Second)

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
	s.setExpire(ctx, 1*time.Minute)

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
		Where("status_finished = ?0 and status_registered = ?0", true)
	query, err = component.OrderByIndex(query, params.Order, pulseNumber, sequenceNumber, params.Index != nil, models.TxIndexTypeFinishPulseRecord)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError(err.Error()))
	}
	err = query.
		Limit(limit).
		Select(&result)
	if err != nil {
		s.log.Error(err)
		s.setExpire(ctx, 1*time.Second)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	if len(result) == 0 {
		return ctx.NoContent(http.StatusNoContent)
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
	s.setExpire(ctx, 1*time.Minute)

	if !isInt(amount) {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("invalid amount"))
	}
	if strings.HasPrefix(amount, "-") {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("negative amount"))
	}

	return ctx.JSON(http.StatusOK, ResponsesFeeYaml{Fee: s.config.FeeAmount.String()})
}

func (s *ObserverServer) Member(ctx echo.Context, reference string) error {
	ref, errMsg := s.checkReference(reference)
	if errMsg != nil {
		if appfoundation.IsEthereumAddress(reference) {
			return s.getMember(ctx, getByMigrationAddress, reference)
		}
		return ctx.JSON(http.StatusBadRequest, *errMsg)
	}
	return s.getMember(ctx, getByReference, ref.String())
}

func (s *ObserverServer) MemberByPublicKey(ctx echo.Context, params MemberByPublicKeyParams) error {
	publicKey, err := foundation.ExtractCanonicalPublicKey(params.PublicKey)
	if err != nil {
		s.log.Error(errors.Wrap(err, fmt.Sprintf("extracting canonical pk failed, current value %v", publicKey)))
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}
	return s.getMember(ctx, getByPublicKey, publicKey)
}

const (
	getByReference = iota
	getByMigrationAddress
	getByPublicKey
)

func (s *ObserverServer) getMember(ctx echo.Context, method int, smth string) error {
	s.setExpire(ctx, 1*time.Second)

	var member *models.Member
	var err error
	var memberReference []byte

	switch method {
	case getByReference:
		ref, refErr := insolar.NewReferenceFromString(smth)
		if refErr != nil {
			panic("invalid reference")
		}
		member, err = component.GetMember(ctx.Request().Context(), s.db, ref.Bytes())
	case getByMigrationAddress:
		member, err = component.GetMemberByMigrationAddress(ctx.Request().Context(), s.db, smth)
	case getByPublicKey:
		member, err = component.GetMemberByPublicKey(ctx.Request().Context(), s.db, smth)
	}

	if member != nil {
		memberReference = member.Reference
	}
	if err != nil {
		if err == component.ErrReferenceNotFound {
			return ctx.NoContent(http.StatusNoContent)
		}
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	pulse, err := s.pStorage.Last()
	if err != nil {
		s.log.Error(errors.Wrap(err, "couldn't load last pulse"))
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	pTime, err := pulse.Number.AsApproximateTime()
	if err != nil {
		s.log.Error(errors.Wrapf(err, "couldn't convert pulse %d to time", pulse.Number))
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	deposits, err := component.GetDeposits(ctx.Request().Context(), s.db, memberReference, true)
	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	response, err := MemberToAPIMember(*member, deposits, pTime.Unix(), method != getByReference)
	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}
	return ctx.JSON(http.StatusOK, response)
}

func (s *ObserverServer) Balance(ctx echo.Context, reference string) error {
	s.setExpire(ctx, 1*time.Second)

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
	s.setExpire(ctx, 10*time.Second)

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

	res := SchemasTransactions{}
	for _, t := range txs {
		res = append(res, TxToAPITx(t, models.TxIndexTypePulseRecord))
	}
	return ctx.JSON(http.StatusOK, res)
}

func (s *ObserverServer) Notification(ctx echo.Context) error {
	s.setExpire(ctx, 1*time.Minute)

	res, err := component.GetNotification(ctx.Request().Context(), s.db)
	if err != nil {
		if err == component.ErrNotificationNotFound {
			return ctx.NoContent(http.StatusNoContent)
		}
		s.log.Error(err)
		s.setExpire(ctx, 1*time.Second)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, ResponsesNotificationInfoYaml{
		Notification: res.Message,
	})
}

func (s *ObserverServer) Transaction(ctx echo.Context, txIDStr string) error {
	s.setExpire(ctx, 10*time.Second)

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
	s.setExpire(ctx, 10*time.Second)

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

func (s *ObserverServer) SupplyStatsTotal(ctx echo.Context) error {
	s.setExpire(ctx, 10*time.Second)

	repo := postgres.NewSupplyStatsRepository(s.db)
	result, err := repo.LastStats()
	if err != nil {
		s.log.Error(errors.Wrap(err, "couldn't get last supply stats"))
		return ctx.JSON(http.StatusInternalServerError, "")
	}

	return ctx.String(http.StatusOK, result.TotalInXNS())
}

// PointsCount holds count of history points. Max count is 21
// https://insolar.atlassian.net/browse/INS-4049
const PointsCount = 21

func (s *ObserverServer) MarketStats(ctx echo.Context) error {
	s.setExpire(ctx, 1*time.Hour)
	switch s.config.PriceOrigin {
	case "binance":
		repo := postgres.NewBinanceStatsRepository(s.db)
		stats, err := repo.LastStats()
		if err != nil {
			s.log.Error(errors.Wrap(err, "couldn't get last binance supply stats"))
			return ctx.JSON(http.StatusInternalServerError, "")
		}

		history, err := repo.PriceHistory(PointsCount)
		if err != nil {
			s.log.Error(errors.Wrap(err, "couldn't get info about price history"))
			return ctx.JSON(http.StatusInternalServerError, "")
		}
		response := ResponsesMarketStatsYaml{
			Price:       fmt.Sprintf("%v", stats.SymbolPriceUSD),
			DailyChange: NullableString(stats.PriceChangePercent),
		}
		response.addHistoryPoints(history)

		return ctx.JSON(http.StatusOK, response)
	case "coin_market_cap":
		repo := postgres.NewCoinMarketCapStatsRepository(s.db)
		stats, err := repo.LastStats()
		if err != nil {
			s.log.Error(errors.Wrap(err, "couldn't get last coin market cap supply stats"))
			return ctx.JSON(http.StatusInternalServerError, "")
		}
		history, err := repo.PriceHistory(PointsCount)
		if err != nil {
			s.log.Error(errors.Wrap(err, "couldn't get info about price history"))
			return ctx.JSON(http.StatusInternalServerError, "")
		}
		response := ResponsesMarketStatsYaml{
			CirculatingSupply: NullableString(fmt.Sprintf("%v", stats.CirculatingSupply)),
			DailyChange:       NullableString(fmt.Sprintf("%v", stats.PercentChange24Hours)),
			MarketCap:         NullableString(fmt.Sprintf("%v", stats.MarketCap)),
			Price:             fmt.Sprintf("%v", stats.Price),
			Rank:              NullableString(fmt.Sprintf("%v", stats.Rank)),
			Volume:            NullableString(fmt.Sprintf("%v", stats.Volume24Hours)),
		}
		response.addHistoryPoints(history)
		return ctx.JSON(http.StatusOK, response)
	default:
		return ctx.JSON(http.StatusOK, ResponsesMarketStatsYaml{
			Price: s.config.Price,
		})
	}
}

func (s *ObserverServer) NetworkStats(ctx echo.Context) error {
	s.setExpire(ctx, 10*time.Second)

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
	st := "registered"
	if status != nil {
		st = *status
	}
	query, err = component.FilterByStatus(query, st)
	if err != nil {
		errorMsg.Error = append(errorMsg.Error, err.Error())
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

func (s *ObserverServer) setExpire(ctx echo.Context, duration time.Duration) {
	ctx.Response().Header().Set(
		"Cache-Control",
		fmt.Sprintf("max-age=%d", int(duration.Seconds())),
	)
	ctx.Response().Header().Set(
		"Expires",
		time.Now().UTC().Add(duration).Format(http.TimeFormat),
	)
}
