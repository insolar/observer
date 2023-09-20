package tree

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer/store"
)

func TestBuilder_Build(t *testing.T) {
	mc := minimock.NewController(t)
	fetcher := store.NewRecordFetcherMock(mc)
	ctx := context.Background()

	reqID := gen.ID()

	fetcher.RequestMock.Return(
		record.Material{
			Virtual: record.Virtual{
				Union: &record.Virtual_IncomingRequest{
					IncomingRequest: &record.IncomingRequest{
						Arguments: []byte{3, 2, 1},
					},
				},
			},
		},
		nil,
	)

	fetcher.ResultMock.Return(
		record.Material{
			Virtual: record.Virtual{
				Union: &record.Virtual_Result{
					Result: &record.Result{Payload: []byte{1, 2, 3}},
				},
			},
		},
		nil,
	)

	fetcher.CalledRequestsMock.Return(
		[]record.Material{},
		nil,
	)

	fetcher.SideEffectMock.Return(
		record.Material{},
		store.ErrNotFound,
	)

	b := NewBuilder(fetcher)
	require.NotNil(t, b)

	tree, err := b.Build(ctx, reqID)
	require.NoError(t, err)

	expected := Structure{
		RequestID: reqID,
		Request: record.IncomingRequest{
			Arguments: []byte{3, 2, 1},
		},
		Result:    record.Result{Payload: []byte{1, 2, 3}},
		Outgoings: []Outgoing{},
	}

	require.Equal(t, expected, tree)
}
