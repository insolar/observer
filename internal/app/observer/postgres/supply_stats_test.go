// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !node

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/testutils"
	"github.com/insolar/observer/observability"

	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
)

func TestSupplyStats(t *testing.T) {
	supplyStatsRepository := postgres.NewSupplyStatsRepository(db)
	depositRepo := postgres.NewDepositStorage(observability.Make(context.Background()), db)

	t.Run("two deposits and one member", func(t *testing.T) {
		defer testutils.TruncateTables(t, db, []interface{}{
			&models.Member{},
			&models.SupplyStats{},
			&models.Deposit{},
		})
		stats, err := supplyStatsRepository.CountStats()
		require.NoError(t, err)
		require.Equal(t, "0", stats.Total)

		memberRef := gen.Reference()
		now := time.Now().Unix()
		member := models.Member{
			Reference: memberRef.Bytes(),
			Balance:   "200",
		}
		err = db.Insert(&member)
		require.NoError(t, err)

		deposit1 := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Member:          memberRef,
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "100",
			DepositState:    gen.ID(),
		}

		deposit2 := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Member:          memberRef,
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "100",
			DepositState:    gen.ID(),
		}

		err = depositRepo.Insert(deposit1)
		require.NoError(t, err, "insert")
		err = depositRepo.Insert(deposit2)
		require.NoError(t, err, "insert")

		stats, err = supplyStatsRepository.CountStats()
		require.NoError(t, err)
		require.Equal(t, "400", stats.Total)

		stats, err = supplyStatsRepository.LastStats()
		require.Error(t, err)

		stats, err = supplyStatsRepository.CountStats()
		require.NoError(t, err)
		err = supplyStatsRepository.InsertStats(stats)
		require.NoError(t, err)

		stats, err = supplyStatsRepository.LastStats()
		require.NoError(t, err)
		require.Equal(t, "400", stats.Total)
	})

	t.Run("one member", func(t *testing.T) {
		defer testutils.TruncateTables(t, db, []interface{}{
			&models.Member{},
			&models.SupplyStats{},
			&models.Deposit{},
		})
		stats, err := supplyStatsRepository.CountStats()
		require.NoError(t, err)
		require.Equal(t, "0", stats.Total)

		member := models.Member{
			Reference: gen.Reference().Bytes(),
			Balance:   "200",
		}
		err = db.Insert(&member)
		require.NoError(t, err)

		stats, err = supplyStatsRepository.CountStats()
		require.NoError(t, err)
		require.Equal(t, "200", stats.Total)

		stats, err = supplyStatsRepository.LastStats()
		require.Error(t, err)

		stats, err = supplyStatsRepository.CountStats()
		require.NoError(t, err)
		err = supplyStatsRepository.InsertStats(stats)
		require.NoError(t, err)

		stats, err = supplyStatsRepository.LastStats()
		require.NoError(t, err)
		require.Equal(t, "200", stats.Total)
	})

	t.Run("two members", func(t *testing.T) {
		defer testutils.TruncateTables(t, db, []interface{}{
			&models.Member{},
			&models.SupplyStats{},
			&models.Deposit{},
		})
		stats, err := supplyStatsRepository.CountStats()
		require.NoError(t, err)
		require.Equal(t, "0", stats.Total)

		member1 := models.Member{
			Reference: gen.Reference().Bytes(),
			Balance:   "200",
		}
		err = db.Insert(&member1)
		require.NoError(t, err)

		member2 := models.Member{
			Reference: gen.Reference().Bytes(),
			Balance:   "200",
		}
		err = db.Insert(&member2)
		require.NoError(t, err)

		stats, err = supplyStatsRepository.CountStats()
		require.NoError(t, err)
		require.Equal(t, "400", stats.Total)

		stats, err = supplyStatsRepository.LastStats()
		require.Error(t, err)

		stats, err = supplyStatsRepository.CountStats()
		require.NoError(t, err)
		err = supplyStatsRepository.InsertStats(stats)
		require.NoError(t, err)

		stats, err = supplyStatsRepository.LastStats()
		require.NoError(t, err)
		require.Equal(t, "400", stats.Total)
	})
}
