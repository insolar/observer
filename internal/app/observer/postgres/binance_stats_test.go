// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres_test

import (
	"testing"
	"time"

	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
	"github.com/insolar/observer/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestBinanceStats(t *testing.T) {
	db, _, dbCleaner := testutils.SetupDB("../../../../scripts/migrations")
	defer dbCleaner()
	repo := postgres.NewBinanceStatsRepository(db)

	_, err := repo.LastStats()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no data")

	stat := models.BinanceStats{
		Symbol:             "TEST",
		SymbolPriceBTC:     "111",
		SymbolPriceUSD:     2222,
		BTCPriceUSD:        "3333",
		PriceChangePercent: "4444",
	}
	err = repo.InsertStats(&stat)
	require.NoError(t, err)

	savedData, err := repo.LastStats()
	require.NoError(t, err)

	require.Equal(t, stat.PriceChangePercent, savedData.PriceChangePercent)
	require.Equal(t, stat.BTCPriceUSD, savedData.BTCPriceUSD)
	require.Equal(t, stat.SymbolPriceUSD, savedData.SymbolPriceUSD)
	require.Equal(t, stat.SymbolPriceBTC, savedData.SymbolPriceBTC)
	require.Equal(t, stat.Symbol, savedData.Symbol)
}

func TestBinanceStats_AverageCalculation(t *testing.T) {
	mockDB, _, dbCleaner := testutils.SetupDB("../../../../scripts/migrations")
	defer dbCleaner()
	repo := postgres.NewBinanceStatsRepository(mockDB)

	saveTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	// 24 hours
	for i := 0; i < 1339; i++ {
		saveTime = saveTime.Add(1 * time.Minute)
		stat := models.BinanceStats{
			Symbol:             "TEST",
			SymbolPriceBTC:     "1",
			SymbolPriceUSD:     1,
			BTCPriceUSD:        "3",
			PriceChangePercent: "4",
			Created:            saveTime,
		}
		err := mockDB.Insert(&stat)
		require.NoError(t, err)
	}

	points, err := repo.PriceHistory(24)
	require.NoError(t, err)
	require.Equal(t, 3, len(points))

	require.Equal(t,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		points[0].Timestamp)
	require.Equal(t,
		time.Date(2020, 1, 1, 8, 0, 0, 0, time.UTC),
		points[1].Timestamp)
	require.Equal(t,
		time.Date(2020, 1, 1, 16, 0, 0, 0, time.UTC),
		points[2].Timestamp)

	require.Equal(t, float64(1), points[0].Price)
	require.Equal(t, float64(1), points[1].Price)
	require.Equal(t, float64(1), points[2].Price)

}
