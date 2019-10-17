package store

import (
	"context"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)


type RecordFetcher interface {
	Request(ctx context.Context, reqID insolar.ID) (record.Material, error)
	Result(ctx context.Context, reqID insolar.ID) (record.Material, error)
	SideEffect(ctx context.Context, reqID insolar.ID) (record.Material, error)
	CalledRequests(ctx context.Context, reqID insolar.ID) ([]record.Material, error)
}

type RecordSetter interface {
	SetResult(ctx context.Context, record record.Material) error
	SetSideEffect(ctx context.Context, record record.Material) error
	SetRequest(ctx context.Context, record record.Material) error
}
