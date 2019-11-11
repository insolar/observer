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
	"github.com/pkg/errors"
)

type NetworkStatsModel struct {
	tableName struct{} `sql:"network_stats"` //nolint: unused,structcheck

	Created           time.Time `sql:"created,pk,default:now(),notnull"`
	PulseNumber       int       `sql:"pulse_number,notnull"`
	TotalTransactions int       `sql:"total_transactions,notnull"`
	MonthTransactions int       `sql:"month_transactions,notnull"`
	TotalAccounts     int       `sql:"total_accounts,notnull"`
	Nodes             int       `sql:"nodes,notnull"`
	CurrentTPS        int       `sql:"current_tps,notnull"`
	MaxTPS            int       `sql:"max_tps,notnull"`
}

//go:generate minimock -i NetworkStatsRepo -o ./ -s _mock.go -g

type NetworkStatsRepo interface {
	LastStats() (NetworkStatsModel, error)
	InsertStats(NetworkStatsModel) error
	CountStats() (NetworkStatsModel, error)
}

type NetworkStatsRepository struct {
	db orm.DB
}

func NewNetworkStatsRepository(db orm.DB) NetworkStatsRepo {
	return &NetworkStatsRepository{db: db}
}

func (s *NetworkStatsRepository) LastStats() (NetworkStatsModel, error) {
	lastStats := &NetworkStatsModel{}
	err := s.db.Model(lastStats).Last()
	if err != nil && err != pg.ErrNoRows {
		return NetworkStatsModel{}, errors.Wrap(err, "failed request to db")
	}
	if err == pg.ErrNoRows {
		return NetworkStatsModel{}, errors.New("No data")
	}
	return *lastStats, nil
}

func (s *NetworkStatsRepository) InsertStats(xcs NetworkStatsModel) error {
	stats := NetworkStatsModel{
		Created: time.Now(),
	}

	err := s.db.Insert(&stats)
	if err != nil {
		return errors.Wrap(err, "failed to insert stats")
	}

	return nil
}

func (s *NetworkStatsRepository) CountStats() (NetworkStatsModel, error) {
	stats := NetworkStatsModel{
		Created: time.Now(),
	}

	// LastPulseNumber and Nodes
	{
		pulseSchema := PulseSchema{}
		err := s.db.Model(&pulseSchema).
			Order("pulse DESC").
			Limit(1).
			Select()
		if err != nil {
			return NetworkStatsModel{}, errors.Wrap(err, "couldn't get last pulse data")
		}
		stats.PulseNumber = int(pulseSchema.Pulse)
		stats.Nodes = int(pulseSchema.Nodes)
	}

	// MonthTransaction
	{
		sqlRes := struct{ Count int }{}
		_, err := s.db.QueryOne(&sqlRes, "SELECT COUNT(1) AS Count FROM simple_transactions"+
			" WHERE finish_pulse_record[0] >= NOW() - INTERVAL '30 DAYS'")
		if err != nil {
			return NetworkStatsModel{}, errors.Wrap(err, "couldn't count total transactions")
		}
		stats.MonthTransactions = sqlRes.Count
	}

	// TotalTransactions
	{
		sqlRes := struct{ Count int }{}
		_, err := s.db.QueryOne(&sqlRes, "SELECT COUNT(1) AS Count FROM simple_transactions")
		if err != nil {
			return NetworkStatsModel{}, errors.Wrap(err, "couldn't count total transactions")
		}
		stats.TotalTransactions = sqlRes.Count
	}

	// TotalAccounts
	{
		sqlRes := struct{ Count int }{}
		_, err := s.db.QueryOne(&sqlRes, "SELECT COUNT(1) AS Count FROM members")
		if err != nil {
			return NetworkStatsModel{}, errors.Wrap(err, "couldn't count total accounts")
		}
		stats.TotalAccounts = sqlRes.Count
	}

	// MaxTPS
	{
		sqlRes := struct{ Count int }{}
		sql := "SELECT MAX(t.tpp) AS Count FROM (SELECT COUNT(1) as tpp FROM" +
			" simple_transactions GROUP BY finish_pulse_record[0]) AS t"
		_, err := s.db.QueryOne(&sqlRes, sql)
		if err != nil {
			return NetworkStatsModel{}, errors.Wrap(err, "failed request to db")
		}
		stats.MaxTPS = sqlRes.Count
	}

	// CurrentTPS
	{
		sqlRes := struct{ Count int }{}
		sql := "SELECT COUNT(1) AS Count FROM simple_transactions" +
			" WHERE finish_pulse_record[0] = (" +
			"   SELECT finish_pulse_record[0] FROM simple_transactions" +
			"   WHERE finish_pulse_record IS NOT NULL ORDER BY id DESC LIMIT 1" +
			" )"
		_, err := s.db.QueryOne(&sqlRes, sql)
		if err != nil {
			return NetworkStatsModel{}, errors.Wrap(err, "failed request to db")
		}
		stats.CurrentTPS = sqlRes.Count
	}

	return stats, nil
}
