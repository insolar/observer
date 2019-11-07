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

type StatsModel struct {
	tableName struct{} `sql:"xns_coin_stats"` //nolint: unused,structcheck

	ID          uint64    `sql:"id,pk"`
	Created     time.Time `sql:"created default:now(),notnull"`
	Total       string    `sql:"total"`
	Max         string    `sql:"max"`
	Circulating string    `sql:"circulating"`
}

//go:generate minimock -i StatsRepo -o ./ -s _mock.go -g

type StatsRepo interface {
	LastStats() (StatsModel, error)
	InsertStats(StatsModel) error
	CountStats(time *time.Time) (StatsModel, error)
}

type StatsRepository struct {
	db orm.DB
}

func NewStatsRepository(db orm.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

func (s *StatsRepository) LastStats() (StatsModel, error) {
	lastStats := &StatsModel{}
	err := s.db.Model(lastStats).Last()
	if err != nil && err != pg.ErrNoRows {
		return StatsModel{}, errors.Wrap(err, "failed request to db")
	}
	if err == pg.ErrNoRows {
		return StatsModel{}, errors.New("No data")
	}
	return *lastStats, nil
}

func (s *StatsRepository) CountStats(collectDT *time.Time) (StatsModel, error) {
	var dt int64
	if collectDT != nil {
		dt = collectDT.Unix()
	} else {
		dt = time.Now().Unix()
	}

	sql := fmt.Sprintf(`
WITH stats as (SELECT d.amount::numeric(24),
                      d.hold_release_date,
                      %d     as pulse_ts,
                      d.vesting
               from deposits d),
     sums as (
         SELECT sum(floor(stats.amount * least(greatest(stats.pulse_ts - stats.hold_release_date, 0) /stats.vesting, 1))) as free
         from stats
         ),
     circulating as (select sum(balance::numeric(24)) as sum from members),
     max_amount as (select sum(amount::numeric(24)) as sum from deposits where eth_hash = 'genesis_deposit')

SELECT
       circulating.sum::text                   as circulating,
       (circulating.sum + free)::numeric             as total,
       (max_amount.sum)::text  as max
from sums,
     circulating,
     max_amount
;
`, dt)
	stats := StatsModel{}
	_, err := s.db.Query(&stats, sql)
	if err != nil {
		return StatsModel{}, errors.Wrap(err, "failed request to db")
	}
	return stats, nil
}

func (s *StatsRepository) InsertStats(xcs StatsModel) error {
	stats := StatsModel{
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
