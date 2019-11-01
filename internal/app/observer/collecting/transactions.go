package collecting

import (
	"context"
	"encoding/json"

	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/insolar/insolar/application/builtin/contract/member/signer"
	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	proxyMember "github.com/insolar/insolar/application/builtin/proxy/member"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/models"
)

const (
	callSiteTransfer  = "member.transfer"
	callSiteMigration = "deposit.migration"
	callSiteRelease   = "deposit.transfer"
)

const (
	methodCall              = "Call"
	methodTransferToDeposit = "TransferToDeposit"
	methodTransfer          = "Transfer"
)

const (
	paramAmount      = "amount"
	paramToMemberRef = "toMemberReference"
)

type TxRegisterCollector struct {
}

func NewTxRegisterCollector() *TxRegisterCollector {
	return &TxRegisterCollector{}
}

func (c *TxRegisterCollector) Collect(ctx context.Context, rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	var tx *observer.TxRegister
	switch request.Method {
	case methodCall:
		tx = registerTransfer(rec)
	case methodTransferToDeposit:
		tx = registerMigration(rec)
	case methodTransfer:
		tx = registerRelease(rec)
	default:
		return nil
	}
	if tx == nil {
		return nil
	}
	if err := tx.Validate(); err != nil {
		log.Error(errors.Wrap(err, "invalid transaction received"))
		return nil
	}
	return tx
}

func registerTransfer(rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	// Skip non-member objects.
	if request.Prototype == nil || !request.Prototype.Equal(*proxyMember.PrototypeReference) {
		return nil
	}

	if request.Method != methodCall {
		return nil
	}

	// Skip internal calls.
	if request.APINode.IsEmpty() {
		return nil
	}

	// Skip saga.
	if request.IsDetachedCall() {
		return nil
	}

	args, callParams, err := parseExternalArguments(request.Arguments)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse arguments"))
		return nil
	}
	if args.Params.CallSite != callSiteTransfer {
		return nil
	}

	memberFrom, err := insolar.NewObjectReferenceFromString(args.Params.Reference)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse from reference"))
		return nil
	}
	toMemberStr, ok := callParams[paramToMemberRef].(string)
	if !ok {
		log.Error(errors.Wrap(err, "failed to parse from reference"))
		return nil
	}
	memberTo, err := insolar.NewObjectReferenceFromString(toMemberStr)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse to reference"))
		return nil
	}
	amount, ok := callParams[paramAmount].(string)
	if !ok {
		log.Error(errors.Wrap(err, "failed to parse from amount"))
		return nil
	}

	return &observer.TxRegister{
		Type:                models.TTypeTransfer,
		TransactionID:       insolar.NewReference(rec.Record.ID).Bytes(),
		PulseNumber:         int64(rec.Record.ID.Pulse()),
		RecordNumber:        int64(rec.RecordNumber),
		Amount:              amount,
		MemberFromReference: memberFrom.Bytes(),
		MemberToReference:   memberTo.Bytes(),
	}
}

func registerMigration(rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	// Skip non-deposit objects.
	if request.Prototype == nil || !request.Prototype.Equal(*proxyDeposit.PrototypeReference) {
		return nil
	}

	if request.Method != methodTransferToDeposit {
		return nil
	}

	// Skip external calls.
	if request.Caller.IsEmpty() {
		return nil
	}

	var (
		amount                                string
		txID, toDeposit, fromMember, toMember insolar.Reference
	)
	err := insolar.Deserialize(request.Arguments, []interface{}{&amount, &toDeposit, &fromMember, &txID, &toMember})
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse arguments"))
		return nil
	}

	return &observer.TxRegister{
		Type:                models.TTypeMigration,
		TransactionID:       txID.Bytes(),
		PulseNumber:         int64(rec.Record.ID.Pulse()),
		RecordNumber:        int64(rec.RecordNumber),
		MemberFromReference: fromMember.Bytes(),
		MemberToReference:   toMember.Bytes(),
		DepositToReference:  toDeposit.Bytes(),
		Amount:              amount,
	}
}

func registerRelease(rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	// Skip non-deposit objects.
	if request.Prototype == nil || !request.Prototype.Equal(*proxyDeposit.PrototypeReference) {
		return nil
	}

	if request.Method != methodTransfer {
		return nil
	}

	// Skip external calls.
	if request.Caller.IsEmpty() {
		return nil
	}

	var (
		amount         string
		txID, toMember insolar.Reference
	)
	err := insolar.Deserialize(request.Arguments, []interface{}{&amount, &toMember, &txID})
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse arguments"))
		return nil
	}

	return &observer.TxRegister{
		Type:                 models.TTypeRelease,
		TransactionID:        txID.Bytes(),
		PulseNumber:          int64(rec.Record.ID.Pulse()),
		RecordNumber:         int64(rec.RecordNumber),
		MemberToReference:    toMember.Bytes(),
		DepositFromReference: insolar.NewReference(rec.Record.ObjectID).Bytes(),
		Amount:               amount,
	}
}

func parseExternalArguments(in []byte) (member.Request, map[string]interface{}, error) {
	if in == nil {
		return member.Request{}, nil, nil
	}
	var signedRequest []byte
	err := insolar.Deserialize(in, []interface{}{&signedRequest})
	if err != nil {
		return member.Request{}, nil, err
	}

	if len(signedRequest) == 0 {
		return member.Request{}, nil, errors.New("failed to parse signed request")
	}
	request := member.Request{}
	{
		var encodedRequest []byte
		// IMPORTANT: argument number should match serialization. This is why we use nil as second and third values.
		err = signer.UnmarshalParams(signedRequest, []interface{}{&encodedRequest, nil, nil}...)
		if err != nil {
			return member.Request{}, nil, errors.Wrapf(err, "failed to unmarshal params")
		}
		err = json.Unmarshal(encodedRequest, &request)
		if err != nil {
			return member.Request{}, nil, errors.Wrapf(err, "failed to unmarshal json member request")
		}
	}

	if request.Params.CallParams == nil {
		return request, nil, nil
	}

	callParams, ok := request.Params.CallParams.(map[string]interface{})
	if !ok {
		return member.Request{}, nil, errors.New("failed to decode CallParams")
	}
	return request, callParams, nil
}

type TxResultCollector struct {
	fetcher store.RecordFetcher
	log     *logrus.Logger
}

func NewTxResultCollector(log *logrus.Logger, fetcher store.RecordFetcher) *TxResultCollector {
	return &TxResultCollector{
		fetcher: fetcher,
		log:     log,
	}
}

func (c *TxResultCollector) Collect(ctx context.Context, rec exporter.Record) *observer.TxResult {
	result, ok := record.Unwrap(&rec.Record.Virtual).(*record.Result)
	if !ok {
		return nil
	}

	txID := result.Request
	requestRecord, err := c.fetcher.Request(ctx, *txID.GetLocal())
	if err != nil {
		c.log.Error(errors.Wrapf(
			err,
			"failed to fetch request with id %s",
			txID.GetLocal().DebugString()),
		)
		return nil
	}

	request, ok := record.Unwrap(&requestRecord.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	if request.Method != methodCall {
		return nil
	}
	// Skip non-API requests.
	if request.APINode.IsEmpty() {
		return nil
	}
	// Skip saga.
	if request.IsDetachedCall() {
		return nil
	}
	args, _, err := parseExternalArguments(request.Arguments)
	if err != nil {
		c.log.Error(errors.Wrap(err, "failed to parse request arguments"))
		return nil
	}

	// Migration and release don't have fees.
	if args.Params.CallSite == callSiteMigration || args.Params.CallSite == callSiteRelease {
		tx := &observer.TxResult{
			TransactionID: txID.Bytes(),
			Fee:           "0",
		}
		if err = tx.Validate(); err != nil {
			c.log.Error(errors.Wrap(err, "failed to validate transaction"))
			return nil
		}
		return tx
	}

	// Processing transfer between members. Its the only transfer that has fee.
	if args.Params.CallSite != callSiteTransfer {
		return nil
	}
	response := member.TransferResponse{}
	err = insolar.Deserialize(result.Payload, &foundation.Result{
		Returns: []interface{}{&response, nil},
	})
	if err != nil {
		log.Error(errors.Wrap(err, "failed to deserialize method result"))
		return nil
	}

	tx := &observer.TxResult{
		TransactionID: txID.Bytes(),
		Fee:           response.Fee,
	}
	if err = tx.Validate(); err != nil {
		c.log.Error(errors.Wrap(err, "failed to validate transaction"))
		return nil
	}
	return tx
}
