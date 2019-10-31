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

package xnscoinstats

import (
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
)

//go:generate minimock -i StatsRepo -o ./ -s _mock.go -g

type StatsRepo interface {
	LastStats() (StatsModel, error)
	InsertStats(xcs XnsCoinStats) error
	CountStats() (XnsCoinStats, error)
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

func (s *StatsRepository) CountStats() (XnsCoinStats, error) {
	sql := `
WITH stats as (SELECT d.amount::numeric(24),
                      d.hold_release_date,
                      round(extract(epoch from now()))      as pulse_ts,
                      d.hold_release_date + 365 * 24 * 3600 as vesting_ends_ts
               from deposits d),
     sums as (
         SELECT sum(floor(stats.amount * least(greatest(stats.pulse_ts - stats.hold_release_date, 0) /
                                               (stats.vesting_ends_ts - stats.hold_release_date), 1)))      as free
         from stats),
     circulating as (select sum(balance::numeric(24)) as sum from members),
     dep_balance as (select sum(balance::numeric(24)) as sum from deposits)

SELECT
       circulating.sum::numeric                   as circulating,
       (circulating.sum + free)::numeric            as total,
       (circulating.sum + dep_balance.sum)::numeric as max
from sums,
     circulating,
     dep_balance
;
`
	stats := XnsCoinStats{}
	_, err := s.db.Query(&stats, sql)
	if err != nil {
		return XnsCoinStats{}, errors.Wrap(err, "failed request to db")
	}
	return stats, nil
}

func (s *StatsRepository) InsertStats(xcs XnsCoinStats) error {
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
