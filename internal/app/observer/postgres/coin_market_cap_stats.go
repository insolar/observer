// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres

import (
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/observer/internal/models"
	"github.com/pkg/errors"
)

type CoinMarketCapStatsRepository struct {
	db orm.DB
}

func NewCoinMarketCapStatsRepository(db orm.DB) *CoinMarketCapStatsRepository {
	return &CoinMarketCapStatsRepository{db: db}
}

func (s *CoinMarketCapStatsRepository) InsertStats(stats *models.CoinMarketCapStats) error {
	stats.Created = time.Now().UTC()

	err := s.db.Insert(stats)
	if err != nil {
		return errors.Wrap(err, "failed to insert stats")
	}

	return nil
}

func (s *CoinMarketCapStatsRepository) LastStats() (models.CoinMarketCapStats, error) {
	lastStats := &models.CoinMarketCapStats{}
	err := s.db.Model(lastStats).Last()
	if err != nil && err != pg.ErrNoRows {
		return models.CoinMarketCapStats{}, errors.Wrap(err, "failed request to db")
	}
	if err == pg.ErrNoRows {
		return models.CoinMarketCapStats{}, errors.New("no data")
	}
	return *lastStats, nil
}

func (s *CoinMarketCapStatsRepository) PriceHistory(pointsCount int) ([]models.PriceHistory, error) {
	history := []models.PriceHistory{}

	_, err := s.db.Query(&history,
		`
			select *
			from (
				 select interval_time as timestamp, price_sum / count as price
				 from coin_market_cap_stats_aggregate
				 order by interval_time desc
				 limit ?
			 ) as res
			order by res.timestamp asc
	`, pointsCount)
	if err != nil {
		return nil, err
	}

	return history, nil
}
