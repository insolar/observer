package replica

import (
	"context"
	"testing"

	"github.com/insolar/insolar/component"
	"github.com/insolar/insolar/ledger/heavy/replica"
	"github.com/stretchr/testify/require"
)

func TestGrpcTransport_Call(t *testing.T) {
	ctx := context.Background()
	trans := replica.NewGRPCTransport(20112)
	trans.(component.Initer).Init(ctx)
	trans.(component.Starter).Start(ctx)

	t.Logf("Me: %s", trans.Me())

	reply, err := trans.Send(ctx, "127.0.0.1:20111", "test.Test", []byte("ping"))
	require.NoError(t, err)
	require.Equal(t, "pong", string(reply))
}
