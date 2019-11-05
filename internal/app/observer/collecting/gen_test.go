package collecting

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"

	"github.com/insolar/observer/internal/app/observer"
)

func makeOutgoingRequest() *observer.Record {
	rec := &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_OutgoingRequest{
				OutgoingRequest: &record.OutgoingRequest{},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeResultWith(requestID insolar.ID, result *foundation.Result) *observer.Record {
	payload, err := insolar.Serialize(result)
	if err != nil {
		panic("failed to serialize result")
	}
	ref := insolar.NewReference(requestID)
	rec := &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_Result{
				Result: &record.Result{
					Request: *ref,
					Payload: payload,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}
