// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package dbconn

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/configuration"
)

func TestConnectionHolder_DB(t *testing.T) {
	cfg := configuration.Observer{}.Default()
	db, err := Connect(cfg.DB)
	require.NoError(t, err)
	require.NotNil(t, db)
}
