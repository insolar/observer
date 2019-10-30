package collecting

import (
	"context"
	"encoding/json"
	"runtime/debug"

	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/insolar/insolar/application/builtin/contract/member/signer"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/log"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/models"
)

const (
	txTransfer  = "member.transfer"
	txMigration = "deposit.migration"
	txRelease   = "deposit.transfer"
)

const (
	methodCall              = "Call"
	methodTransferToDeposit = "TransferToDeposit"
	methodTransfer          = "Transfer"
)

type TxRegisterCollector struct {
}

func (c *TxRegisterCollector) Collect(ctx context.Context, rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	switch request.Method {
	case methodCall:
		return collectTransfer(ctx, rec)
	case methodTransferToDeposit:
		return collectMigration(ctx, rec)
	case methodTransfer:
		return collectRelease(ctx, rec)
	}
	log.Error("unknown method: ", request.Method)
	return nil
}

func collectTransfer(ctx context.Context, rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	// Skip internal calls.
	if request.APINode.IsEmpty() {
		return nil
	}

	if request.IsDetachedCall() {
		return nil
	}

	args, callParams := parseExternalArguments(request.Arguments)
	if args.Params.CallSite != txTransfer {
		return nil
	}

	memberFrom, err := insolar.NewObjectReferenceFromString(args.Params.Reference)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse from reference"))
		return nil
	}
	memberTo, err := insolar.NewObjectReferenceFromString(callParams.ToMemberReference)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse to reference"))
		return nil
	}

	return &observer.TxRegister{
		Type:                models.TTypeTransfer,
		TransactionID:       insolar.NewReference(rec.Record.ID).Bytes(),
		PulseNumber:         int64(rec.Record.ID.Pulse()),
		RecordNumber:        int64(rec.RecordNumber),
		Amount:              callParams.Amount,
		MemberFromReference: memberFrom.Bytes(),
		MemberToReference:   memberTo.Bytes(),
	}
}

func collectMigration(ctx context.Context, rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	// Skip external calls.
	if request.Caller.IsEmpty() {
		return nil
	}

	args := parseInternalArguments(request.Arguments)
	if len(args) < 5 {
		log.Error("not enough call params")
		return nil
	}
	amount, ok := args[0].(string)
	if !ok {
		log.Error("failed to parse amount")
		return nil
	}
	toDeposit, ok := args[1].(insolar.Reference)
	if !ok {
		log.Error("failed to parse toDeposit")
		return nil
	}
	fromMember, ok := args[2].(insolar.Reference)
	if !ok {
		log.Error("failed to parse fromMember")
		return nil
	}
	txID, ok := args[3].(insolar.Reference)
	if !ok {
		log.Error("failed to parse txID")
		return nil
	}
	toMember, ok := args[4].(insolar.Reference)
	if !ok {
		log.Error("failed to parse toMember")
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

func collectRelease(ctx context.Context, rec exporter.Record) *observer.TxRegister {
	request, ok := record.Unwrap(&rec.Record.Virtual).(*record.IncomingRequest)
	if !ok {
		return nil
	}

	// Skip external calls.
	if request.Caller.IsEmpty() {
		return nil
	}

	args := parseInternalArguments(request.Arguments)
	if len(args) < 3 {
		log.Error("not enough call params")
		return nil
	}
	amount, ok := args[0].(string)
	if !ok {
		log.Error("failed to parse amount")
		return nil
	}
	toMember, ok := args[1].(insolar.Reference)
	if !ok {
		log.Error("failed to parse toMember")
		return nil
	}
	txID, ok := args[2].(insolar.Reference)
	if !ok {
		log.Error("failed to parse txID")
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

type externalCallParams struct {
	Amount            string `json:"amount"`
	ToMemberReference string `json:"toMemberReference"`
}

func parseInternalArguments(in []byte) []interface{} {
	var args []interface{}
	err := insolar.Deserialize(in, &args)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to parse arguments"))
		return nil
	}
	return args
}

func parseExternalArguments(in []byte) (member.Request, externalCallParams) {
	if in == nil {
		return member.Request{}, externalCallParams{}
	}
	var args []interface{}
	err := insolar.Deserialize(in, &args)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to deserialize request arguments"))
		return member.Request{}, externalCallParams{}
	}

	request := member.Request{}
	if len(args) > 0 {
		if rawRequest, ok := args[0].([]byte); ok {
			var (
				pulseTimeStamp int64
				signature      string
				raw            []byte
			)
			err = signer.UnmarshalParams(rawRequest, &raw, &signature, &pulseTimeStamp)
			if err != nil {
				log.Error(errors.Wrapf(err, "failed to unmarshal params"))
				return member.Request{}, externalCallParams{}
			}
			err = json.Unmarshal(raw, &request)
			if err != nil {
				log.Error(errors.Wrapf(err, "failed to unmarshal json member request"))
				return member.Request{}, externalCallParams{}
			}
		}
	}

	callParams := externalCallParams{}
	data, err := json.Marshal(request.Params.CallParams)
	if err != nil {
		log.Error("failed to marshal CallParams")
		debug.PrintStack()
	}
	err = json.Unmarshal(data, &callParams)
	if err != nil {
		log.Error("failed to unmarshal CallParams")
		debug.PrintStack()
	}
	return request, callParams
}
