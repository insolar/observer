// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package connectivity

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/observability"
)

func TestConnectivity_GRPC(t *testing.T) {
	cfg := configuration.Observer{}.Default()

	t.Run("tls", func(t *testing.T) {
		var conn *Connectivity
		cfg.Replicator.InsecureTLS = false
		conn = Make(cfg, observability.Make(context.Background()))
		require.NotNil(t, conn.GRPC())
	})

	t.Run("insecure", func(t *testing.T) {
		var conn *Connectivity
		cfg.Replicator.InsecureTLS = true
		conn = Make(cfg, observability.Make(context.Background()))
		require.NotNil(t, conn.GRPC())
	})
}

func TestConnectivity_PG(t *testing.T) {
	var conn *Connectivity
	conn = Make(configuration.Observer{}.Default(), observability.Make(context.Background()))
	require.NotNil(t, conn.PG())
}
