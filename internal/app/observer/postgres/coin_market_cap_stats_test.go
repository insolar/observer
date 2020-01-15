//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package postgres_test

import (
	"testing"
	"time"

	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
	"github.com/stretchr/testify/require"
)

func TestCoinMarketCapStats(t *testing.T) {
	repo := postgres.NewCoinMarketCapStatsRepository(db)

	_, err := repo.LastStats()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no data")

	stat := models.CoinMarketCapStats{
		Price:                1,
		PercentChange24Hours: 2,
		Rank:                 3,
		MarketCap:            4,
		Volume24Hours:        5,
		CirculatingSupply:    6,
	}
	err = repo.InsertStats(&stat)
	require.NoError(t, err)

	savedData, err := repo.LastStats()
	require.NoError(t, err)

	require.Equal(t, stat.Price, savedData.Price)
	require.Equal(t, stat.PercentChange24Hours, savedData.PercentChange24Hours)
	require.Equal(t, stat.Rank, savedData.Rank)
	require.Equal(t, stat.MarketCap, savedData.MarketCap)
	require.Equal(t, stat.Volume24Hours, savedData.Volume24Hours)
	require.Equal(t, stat.CirculatingSupply, savedData.CirculatingSupply)
}

func TestCoinMarketCapStats_AverageCalculation(t *testing.T) {
	repo := postgres.NewCoinMarketCapStatsRepository(db)
	saveTime := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	// 24 hours
	for i := 0; i < 1339; i++ {
		saveTime = saveTime.Add(1 * time.Minute)
		stat := models.CoinMarketCapStats{
			Price:                1,
			PercentChange24Hours: 2,
			Rank:                 3,
			MarketCap:            4,
			Volume24Hours:        5,
			CirculatingSupply:    6,
			Created:              saveTime,
		}
		err := db.Insert(&stat)
		require.NoError(t, err)
	}

	points, err := repo.PriceHistory(24)
	require.NoError(t, err)
	require.Equal(t, 3, len(points))

	require.Equal(t,
		time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		points[0].Timestamp)
	require.Equal(t,
		time.Date(2030, 1, 1, 8, 0, 0, 0, time.UTC),
		points[1].Timestamp)
	require.Equal(t,
		time.Date(2030, 1, 1, 16, 0, 0, 0, time.UTC),
		points[2].Timestamp)

	require.Equal(t, float64(1), points[0].Price)
	require.Equal(t, float64(1), points[1].Price)
	require.Equal(t, float64(1), points[2].Price)
}
