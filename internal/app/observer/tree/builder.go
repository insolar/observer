package tree

import (
	"context"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"

	"github.com/insolar/observer/internal/app/observer/store"
)

type Outgoing struct {
	OutgoingRequest record.OutgoingRequest
	Structure       *Structure
	// Result of outgoing, sagas may have this empty
	Result *record.Result
}

type SideEffect struct {
	// one is not nil
	Activation   *record.Activate
	Amend        *record.Amend
	Deactivation *record.Deactivate
}

type Structure struct {
	Request   record.IncomingRequest
	Outgoings []Outgoing

	// SideEffect is optional
	SideEffect *SideEffect

	Result record.Result
}

type Builder interface {
	Build(ctx context.Context, reqID insolar.ID) (Structure, error)
}

type builder struct {
	fetcher store.RecordFetcher //nolint: unused,structcheck
}

func NewBuilder(fetcher store.RecordFetcher) Builder {
	return &builder{}
}

func (b *builder) Build(ctx context.Context, reqID insolar.ID) (Structure, error) {
	return Structure{}, nil
}
