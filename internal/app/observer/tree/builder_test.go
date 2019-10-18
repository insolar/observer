package tree

import (
	"context"
	"testing"

	"github.com/go-pg/pg"
	"github.com/gojuno/minimock"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer/store"
)

func TestBuilder_Build(t *testing.T) {
	mc := minimock.NewController(t)
	store := store.NewRecordFetcherMock(mc)
	ctx := context.Background()

	reqID := gen.ID()

	store.RequestMock.Return(
		record.Material{
			Virtual: record.Virtual{
				Union: &record.Virtual_IncomingRequest{
					IncomingRequest: &record.IncomingRequest{
						Arguments:[]byte{3,2,1},
					},
				},
			},
		},
		nil,
	)

	store.ResultMock.Return(
		record.Material{
			Virtual: record.Virtual{
				Union: &record.Virtual_Result{
					Result: &record.Result{Payload:[]byte{1,2,3}},
				},
			},
		},
		nil,
	)

	store.CalledRequestsMock.Return(
		[]record.Material{},
		nil,
	)

	store.SideEffectMock.Return(
		record.Material{},
		pg.ErrNoRows,
	)

	b := NewBuilder(store)
	require.NotNil(t, b)

	tree, err := b.Build(ctx, reqID)
	require.NoError(t, err)

	expected := Structure{
		RequestID: reqID,
		Request: record.IncomingRequest{
			Arguments:[]byte{3,2,1},
		},
		Result: record.Result{Payload:[]byte{1,2,3}},
		Outgoings: []Outgoing{},
	}

	require.Equal(t, expected, tree)
}
