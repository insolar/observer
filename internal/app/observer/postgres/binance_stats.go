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

type BinanceStatsRepository struct {
	db orm.DB
}

func NewBinanceStatsRepository(db orm.DB) *BinanceStatsRepository {
	return &BinanceStatsRepository{db: db}
}

func (s *BinanceStatsRepository) InsertStats(stats *models.BinanceStats) error {
	stats.Created = time.Now().UTC()

	err := s.db.Insert(stats)
	if err != nil {
		return errors.Wrap(err, "failed to insert stats")
	}

	return nil
}

func (s *BinanceStatsRepository) LastStats() (models.BinanceStats, error) {
	lastStats := &models.BinanceStats{}
	err := s.db.Model(lastStats).Last()
	if err != nil && err != pg.ErrNoRows {
		return models.BinanceStats{}, errors.Wrap(err, "failed request to db")
	}
	if err == pg.ErrNoRows {
		return models.BinanceStats{}, errors.New("no data")
	}
	return *lastStats, nil
}

func (s *BinanceStatsRepository) PriceHistory(pointsCount int) ([]models.PriceHistory, error) {
	history := []models.PriceHistory{}

	_, err := s.db.Query(&history,
		`
				select interval_time as timestamp, price_sum / count as price
				from binance_stats_aggregate
				order by interval_time
				limit ?;
	`, pointsCount)
	if err != nil {
		return nil, err
	}

	return history, nil
}
