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
	stats.Created = time.Now()

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
