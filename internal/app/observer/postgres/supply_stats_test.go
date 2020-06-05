// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !node

package postgres_test

import (
	"testing"

	"github.com/insolar/observer/internal/testutils"

	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
)

func TestSupplyStats(t *testing.T) {
	defer testutils.TruncateTables(t, db, []interface{}{
		&models.Member{},
		&models.SupplyStats{},
	})
	repo := postgres.NewSupplyStatsRepository(db)

	stats, err := repo.CountStats()
	require.NoError(t, err)
	require.Equal(t, "0", stats.Total)

	member := models.Member{
		Reference: gen.Reference().Bytes(),
		Balance:   "1234567890",
	}
	err = db.Insert(&member)
	require.NoError(t, err)

	stats, err = repo.CountStats()
	require.NoError(t, err)
	require.Equal(t, "1234567890", stats.Total)

	stats, err = repo.LastStats()
	require.Error(t, err)

	stats, err = repo.CountStats()
	require.NoError(t, err)
	err = repo.InsertStats(stats)
	require.NoError(t, err)

	stats, err = repo.LastStats()
	require.NoError(t, err)
	require.Equal(t, "1234567890", stats.Total)
}
