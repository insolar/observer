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
	Structure         *Structure
	// Result of outgoing (nil if detached)
	Result *record.Result
}

type SideEffect struct {
	// one is not nil
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
	fetcher store.RecordFetcher //nolint: unused,structcheck
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
	virtualRequest, ok := materialRequest.Virtual.Union.(*record.Virtual_IncomingRequest)
	if !ok {
		return Structure{}, errors.New("not an incoming request")
	}
	tree.Request = *virtualRequest.IncomingRequest

	materialResult, err := b.fetcher.Result(ctx, reqID)
	if err != nil {
		return Structure{}, errors.Wrap(err, "couldn't get result")
	}
	virtualResult, ok := materialResult.Virtual.Union.(*record.Virtual_Result)
	if !ok {
		return Structure{}, errors.New("not a result")
	}
	tree.Result = *virtualResult.Result

	called, err := b.fetcher.CalledRequests(ctx, reqID)
	if err != nil {
		return Structure{}, errors.Wrap(err, "couldn't get outgoings")
	}

	outgoings := make([]Outgoing, 0)
	for _, e := range called {
		switch req := e.Virtual.Union.(type) {
		case *record.Virtual_OutgoingRequest:
			index := int(req.OutgoingRequest.Nonce)
			if add := index - len(outgoings); add > 0 {
				outgoings = append(outgoings, make([]Outgoing, add)...)
			}
			index--

			outgoings[index].OutgoingRequestID = e.ID
			outgoings[index].OutgoingRequest = *req.OutgoingRequest

			result, err := b.fetcher.Result(ctx, e.ID)
			if err != nil {
				return Structure{}, errors.Wrap(err, "couldn't get result")
			}
			outgoings[index].Result = result.Virtual.Union.(*record.Virtual_Result).Result
		case *record.Virtual_IncomingRequest:
			index := int(req.IncomingRequest.Nonce)
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
		return Structure{}, errors.Wrap(err, "couldn't build sub-tree")
	} else {
		switch rec := sideEffect.Virtual.Union.(type) {
		case *record.Virtual_Activate:
			tree.SideEffect = &SideEffect{Activation: rec.Activate}
		case *record.Virtual_Amend:
			tree.SideEffect = &SideEffect{Amend: rec.Amend}
		case *record.Virtual_Deactivate:
			tree.SideEffect = &SideEffect{Deactivation: rec.Deactivate}
		default:
			return Structure{}, errors.New("unexpected side effect")
		}
	}

	return tree, nil
}
