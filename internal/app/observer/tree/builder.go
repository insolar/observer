package tree

import (
	"context"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer/store"
)

type Outgoing struct {
	OutgoingRequestID insolar.ID
	OutgoingRequest   record.OutgoingRequest
	// can be nil in case of detached requests (Saga)
	Structure *Structure
	// Result of outgoing (nil if detached)
	Result *record.Result
}

type SideEffect struct {
	ID insolar.ID
	// one is not nil
	ID           insolar.ID
	Activation   *record.Activate
	Amend        *record.Amend
	Deactivation *record.Deactivate
}

type Structure struct {
	RequestID insolar.ID
	Request   record.IncomingRequest
	Outgoings []Outgoing

	// SideEffect is optional
	SideEffect *SideEffect

	Result record.Result
}

//go:generate minimock -i github.com/insolar/observer/internal/app/observer/tree.Builder -o ./ -s _mock.go -g
type Builder interface {
	Build(ctx context.Context, reqID insolar.ID) (Structure, error)
}

type builder struct {
	fetcher store.RecordFetcher
}

func NewBuilder(fetcher store.RecordFetcher) Builder {
	return &builder{fetcher: fetcher}
}

func (b *builder) Build(ctx context.Context, reqID insolar.ID) (Structure, error) {
	tree := Structure{}
	tree.RequestID = reqID

	materialRequest, err := b.fetcher.Request(ctx, reqID)
	if err != nil {
		return Structure{}, errors.Wrap(err, "couldn't get request")
	}
	virtualRequest, ok := record.Unwrap(&materialRequest.Virtual).(*record.IncomingRequest)
	if !ok {
		return Structure{}, errors.New("not an incoming request")
	}
	tree.Request = *virtualRequest

	materialResult, err := b.fetcher.Result(ctx, reqID)
	if err != nil {
		return Structure{}, errors.Wrap(err, "couldn't get result")
	}
	virtualResult, ok := record.Unwrap(&materialResult.Virtual).(*record.Result)
	if !ok {
		return Structure{}, errors.New("not a result")
	}
	tree.Result = *virtualResult

	called, err := b.fetcher.CalledRequests(ctx, reqID)
	if err != nil {
		return Structure{}, errors.Wrap(err, "couldn't get outgoings")
	}

	outgoings := make([]Outgoing, 0)
	for _, e := range called {
		switch req := record.Unwrap(&e.Virtual).(type) {
		case *record.OutgoingRequest:
			index := int(req.Nonce)
			if add := index - len(outgoings); add > 0 {
				outgoings = append(outgoings, make([]Outgoing, add)...)
			}
			index--

			outgoings[index].OutgoingRequestID = e.ID
			outgoings[index].OutgoingRequest = *req

			result, err := b.fetcher.Result(ctx, e.ID)
			if err != nil {
				if errors.Cause(err) != store.ErrNotFound {
					return Structure{}, errors.Wrap(err, "couldn't get result of outgoing")
				}
			} else {
				outgoings[index].Result = record.Unwrap(&result.Virtual).(*record.Result)
			}
		case *record.IncomingRequest:
			index := int(req.Nonce)
			if add := index - len(outgoings); add > 0 {
				outgoings = append(outgoings, make([]Outgoing, add)...)
			}
			index--

			subTree, err := b.Build(ctx, e.ID)
			if err != nil {
				return Structure{}, errors.Wrap(err, "couldn't build sub-tree")
			}

			outgoings[index].Structure = &subTree
		default:
			panic("unexpected")
		}
	}
	tree.Outgoings = outgoings

	sideEffect, err := b.fetcher.SideEffect(ctx, reqID)
	if err != nil {
		if errors.Cause(err) != store.ErrNotFound {
			return Structure{}, errors.Wrap(err, "couldn't get sub-tree")
		}
	} else {
		switch rec := record.Unwrap(&sideEffect.Virtual).(type) {
		case *record.Activate:
			tree.SideEffect = &SideEffect{ID: sideEffect.ID, Activation: rec}
		case *record.Amend:
			tree.SideEffect = &SideEffect{ID: sideEffect.ID, Amend: rec}
		case *record.Deactivate:
			tree.SideEffect = &SideEffect{ID: sideEffect.ID, Deactivation: rec}
		default:
			return Structure{}, errors.New("unexpected side effect")
		}
	}

	return tree, nil
}
