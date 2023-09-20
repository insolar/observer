package store

import (
	"context"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)

//go:generate minimock -i github.com/insolar/observer/internal/app/observer/store.RecordFetcher -o ./ -s _mock.go -g

type RecordFetcher interface {
	Request(ctx context.Context, reqID insolar.ID) (record.Material, error)
	Result(ctx context.Context, reqID insolar.ID) (record.Material, error)
	SideEffect(ctx context.Context, reqID insolar.ID) (record.Material, error)
	CalledRequests(ctx context.Context, reqID insolar.ID) ([]record.Material, error)
}

//go:generate minimock -i github.com/insolar/observer/internal/app/observer/store.RecordSetter -o ./ -s _mock.go -g

type RecordSetter interface {
	SetResult(ctx context.Context, record record.Material) error
	SetSideEffect(ctx context.Context, record record.Material) error
	SetRequest(ctx context.Context, record record.Material) error
	SetRequestBatch(ctx context.Context, requestRecords []record.Material) error
	SetResultBatch(ctx context.Context, requestRecords []record.Material) error
	SetSideEffectBatch(ctx context.Context, requestRecords []record.Material) error
}

//go:generate minimock -i github.com/insolar/observer/internal/app/observer/store.RecordStore -o ./ -s _mock.go -g

type RecordStore interface {
	RecordFetcher
	RecordSetter
}
