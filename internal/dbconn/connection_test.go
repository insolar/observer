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
