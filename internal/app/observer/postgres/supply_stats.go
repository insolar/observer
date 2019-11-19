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
)

type SupplyStatsModel struct {
	tableName struct{} `sql:"supply_stats"` //nolint: unused,structcheck

	ID          uint64    `sql:"id,pk"`
	Created     time.Time `sql:"created default:now(),notnull"`
	Total       string    `sql:"total"`
	Max         string    `sql:"max"`
	Circulating string    `sql:"circulating"`
}

//go:generate minimock -i SupplyStatsRepo -o ./ -s _mock.go -g

type SupplyStatsRepo interface {
	LastStats() (SupplyStatsModel, error)
	InsertStats(SupplyStatsModel) error
	CountStats(time *time.Time) (SupplyStatsModel, error)
}

type SupplyStatsRepository struct {
	db orm.DB
}

func NewSupplyStatsRepository(db orm.DB) *SupplyStatsRepository {
	return &SupplyStatsRepository{db: db}
}

func (s *SupplyStatsRepository) LastStats() (SupplyStatsModel, error) {
	lastStats := &SupplyStatsModel{}
	err := s.db.Model(lastStats).Last()
	if err != nil && err != pg.ErrNoRows {
		return SupplyStatsModel{}, errors.Wrap(err, "failed request to db")
	}
	if err == pg.ErrNoRows {
		return SupplyStatsModel{}, errors.New("No data")
	}
	return *lastStats, nil
}

func (s *SupplyStatsRepository) CountStats(collectDT *time.Time) (SupplyStatsModel, error) {
	var dt int64
	if collectDT != nil {
		dt = collectDT.Unix()
	} else {
		dt = time.Now().Unix()
	}

	sql := fmt.Sprintf(`
WITH stats as (SELECT d.amount::numeric(24),
                      d.hold_release_date,
                      d.vesting,
                      d.vesting_step,
                      ceil(vesting / vesting_step::float) as steps,
                      case
                          when %d - d.hold_release_date >= vesting
                              THEN ceil(vesting / vesting_step::float)
                          WHEN %d - d.hold_release_date < vesting
                              THEN greatest(%d - d.hold_release_date, 0) / least(d.vesting_step, d.vesting)
                          END
                       as steps_gone
               from deposits d),
     sums as (
         SELECT coalesce(sum(floor(stats.amount * least(steps_gone / steps, 1))), 0) as free
         from stats
         ),
     circulating as (select coalesce(sum(balance::numeric(24)), 0) as sum from members),
     max_amount as (select coalesce(sum(amount::numeric(24)), 0) as sum from deposits where eth_hash = 'genesis_deposit')

SELECT
       circulating.sum::text as circulating,
       (circulating.sum::numeric + free::numeric)::text as total,
       coalesce(max_amount.sum, 0)::text as max
from sums,
     circulating,
     max_amount
;
`, dt, dt, dt)
	stats := SupplyStatsModel{}
	_, err := s.db.Query(&stats, sql)
	if err != nil {
		return SupplyStatsModel{}, errors.Wrap(err, "failed request to db")
	}
	return stats, nil
}

func (s *SupplyStatsRepository) InsertStats(xcs SupplyStatsModel) error {
	stats := SupplyStatsModel{
		Created:     time.Now(),
		Total:       xcs.Total,
		Max:         xcs.Max,
		Circulating: xcs.Circulating,
	}

	err := s.db.Insert(&stats)
	if err != nil {
		return errors.Wrap(err, "failed to insert stats")
	}

	return nil
}
