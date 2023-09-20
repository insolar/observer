package collecting

import (
	"context"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	proxyDeposit "github.com/insolar/mainnet/application/builtin/proxy/deposit"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer"
)

type DepositMemberCollector struct {
	log insolar.Logger
}

func NewDepositMemberCollector(log insolar.Logger) *DepositMemberCollector {
	return &DepositMemberCollector{
		log: log,
	}
}

func (c *DepositMemberCollector) Collect(ctx context.Context, rec *observer.Record) *observer.DepositMemberUpdate {
	if rec == nil {
		return nil
	}

	log := c.log.WithField("recordID", rec.ID.String()).WithField("collector", "DepositMemberCollector")

	req := rec.Virtual.GetIncomingRequest()
	if req == nil {
		log.Debug("not an incoming request, skipping")
		return nil
	}
	if !isConfirmCall(req) {
		log.Debug("not a deposit confirm call, skipping")
		return nil
	}

	if len(req.Arguments) == 0 {
		log.Panic("empty arguments for confirm call")
	}

	var memberRef insolar.Reference
	err := insolar.Deserialize(req.Arguments, []interface{}{nil, nil, nil, nil, &memberRef})
	if err != nil {
		panic(errors.Wrap(err, "couldn't parse arguments"))
	}

	if memberRef.IsEmpty() {
		panic("empty member in Confirm call")
	}

	log.Debugf("update %s: member %s", req.Object.String(), memberRef.String())

	return &observer.DepositMemberUpdate{
		Ref:    *req.Object,
		Member: memberRef,
	}
}

func isConfirmCall(req *record.IncomingRequest) bool {
	if req.Method != "Confirm" || req.CallType != record.CTMethod {
		return false
	}
	return req.Prototype.Equal(*proxyDeposit.PrototypeReference)
}
