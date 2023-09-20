package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/insolar/mainnet/application/appfoundation"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/models"
)

type ObserverServer struct {
	db       *pg.DB
	log      insolar.Logger
	pStorage observer.PulseStorage
	config   configuration.APIConfig
}

func NewObserverServer(db *pg.DB, log insolar.Logger, pStorage observer.PulseStorage, config configuration.APIConfig) *ObserverServer {
	return &ObserverServer{db: db, log: log, pStorage: pStorage, config: config}
}

func (s *ObserverServer) PulseNumber(ctx echo.Context) error {
	pulse, err := s.pStorage.Last()
	if err != nil {
		s.log.Error(errors.Wrap(err, "couldn't load last pulse"))
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, ResponsesPulseNumberYaml{
		PulseNumber: int64(pulse.Number),
	})
}

func (s *ObserverServer) PulseRange(ctx echo.Context, params PulseRangeParams) error {
	limit := params.Limit
	if limit <= 0 || limit > 1000 {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("`limit` should be in range [1, 1000]"))
	}

	if params.FromTimestamp > params.ToTimestamp {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("Invalid input range: fromTimestamp must chronologically precede toTimestamp"))
	}

	pulses, err := s.pStorage.GetRange(params.FromTimestamp, params.ToTimestamp, limit, params.PulseNumber)
	if err != nil {
		s.log.Error(errors.Wrap(err, "couldn't load pulses in range"))
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	if pulses == nil {
		return ctx.NoContent(http.StatusNoContent)
	}

	var res ResponsesPulseRangeYaml
	for _, p := range pulses {
		res = append(res, int64(p))
	}
	return ctx.JSON(http.StatusOK, res)
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

	deposits, err := component.GetDeposits(ctx.Request().Context(), s.db, memberReference, true)
	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	var burnedBalance *models.BurnedBalance
	if insolar.NewReferenceFromBytes(member.Reference).Equal(appfoundation.GetMigrationAdminMember()) {
		burnedBalance, err = component.GetBurnedBalance(s.db)
		if err != nil {
			s.log.Error(err)
			return ctx.JSON(http.StatusInternalServerError, struct{}{})
		}
	}

	response, err := MemberToAPIMember(*member, deposits, burnedBalance, method != getByReference)
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

func (s *ObserverServer) TransactionsByPulseNumberRange(ctx echo.Context, params TransactionsByPulseNumberRangeParams) error {
	s.setExpire(ctx, 10*time.Second)
	var ref *insolar.Reference
	var errMsg *ErrorMessage

	if params.MemberReference != nil {
		ref, errMsg = s.checkReference(*params.MemberReference)
		if errMsg != nil {
			return ctx.JSON(http.StatusBadRequest, *errMsg)
		}
	}

	limit := params.Limit
	if limit <= 0 || limit > 1000 {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("`limit` should be in range [1, 1000]"))
	}

	if params.FromPulseNumber > params.ToPulseNumber {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("Invalid input range: fromPulseNumber must chronologically precede toPulseNumber"))
	}

	var errorMsg ErrorMessage

	var txs []models.Transaction
	var err error
	query := s.db.Model(&txs)

	if ref != nil {
		direction := "all"
		query, err = component.FilterByMemberReferenceAndDirection(query, ref, &direction)
		if err != nil {
			errorMsg.Error = append(errorMsg.Error, err.Error())
		}
	}

	query, err = component.FilterByPulse(query, params.FromPulseNumber, params.ToPulseNumber)
	if err != nil {
		errorMsg.Error = append(errorMsg.Error, err.Error())
	}

	order := "chronological"
	s.getTransactionsOrderByIndex(query, errMsg, params.Index, &order)

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

	return s.getTransactionsOrderByIndex(query, errorMsg, index, order)
}

func (s *ObserverServer) getTransactionsOrderByIndex(
	query *orm.Query, errorMsg *ErrorMessage, index, order *string,
) *orm.Query {
	var err error
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
