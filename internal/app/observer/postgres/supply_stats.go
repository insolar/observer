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
	"fmt"
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/models"
)

type SupplyStatsRepository struct {
	db orm.DB
}

func NewSupplyStatsRepository(db orm.DB) *SupplyStatsRepository {
	return &SupplyStatsRepository{db: db}
}

func (s *SupplyStatsRepository) LastStats() (models.SupplyStats, error) {
	lastStats := &models.SupplyStats{}
	err := s.db.Model(lastStats).Last()
	if err != nil && err != pg.ErrNoRows {
		return models.SupplyStats{}, errors.Wrap(err, "failed request to db")
	}
	if err == pg.ErrNoRows {
		return models.SupplyStats{}, errors.New("No data")
	}
	return *lastStats, nil
}

func (s *SupplyStatsRepository) CountStats() (models.SupplyStats, error) {
	sql := fmt.Sprintf(`select coalesce(sum(balance::numeric(24)), 0) as total from members`)
	stats := models.SupplyStats{}
	_, err := s.db.Query(&stats, sql)
	if err != nil {
		return models.SupplyStats{}, errors.Wrap(err, "failed request to db")
	}
	return stats, nil
}

func (s *SupplyStatsRepository) InsertStats(stats models.SupplyStats) error {
	stats.Created = time.Now()

	err := s.db.Insert(&stats)
	if err != nil {
		return errors.Wrap(err, "failed to insert stats")
	}

	return nil
}
