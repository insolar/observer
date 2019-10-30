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
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/models"
)

const (
	callSiteTransfer = "member.transfer"
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
		tx = collectTransfer(rec)
	case methodTransferToDeposit:
		tx = collectMigration(rec)
	case methodTransfer:
		tx = collectRelease(rec)
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

func collectTransfer(rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	// Skip non-member objects.
	if !request.Prototype.Equal(*proxyMember.PrototypeReference) {
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

func collectMigration(rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	// Skip non-deposit objects.
	if !request.Prototype.Equal(*proxyDeposit.PrototypeReference) {
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

func collectRelease(rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	// Skip non-deposit objects.
	if !request.Prototype.Equal(*proxyDeposit.PrototypeReference) {
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

	callParams, ok := request.Params.CallParams.(map[string]interface{})
	if !ok {
		return member.Request{}, nil, errors.New("failed to decode CallParams")
	}
	return request, callParams, nil
}
