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
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/pkg/cycle"
	"github.com/insolar/observer/observability"
)

type StatisticSchema struct {
	tableName struct{} `sql:"blockchain_stats"`

	Pulse              insolar.PulseNumber `sql:"pulse_num, pk"`
	Transfers          int                 `sql:"count_transactions, notnull"`
	TotalTransfers     int                 `sql:"total_transactions, notnull"`
	TotalMembers       int                 `sql:"total_accounts, notnull"`
	MaxTransfers       int                 `sql:"max_transactions, notnull"`
	LastMonthTransfers int                 `sql:"last_month_transactions, notnull"`
	Nodes              int                 `sql:",notnull"`
}

type StatisticStorage struct {
	cfg          *configuration.Configuration
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewStatisticStorage(cfg *configuration.Configuration, obs *observability.Observability, db orm.DB) *StatisticStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_statistic_storage_error_counter",
		Help: "",
	})
	return &StatisticStorage{
		cfg:          cfg,
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *StatisticStorage) Insert(model *observer.Statistic) error {
	if model == nil {
		s.log.Warnf("trying to insert nil statistic model")
		return nil
	}
	row := statisticSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert()

	if err != nil {
		panic("=== PANIC === BLOCKCHAIN_STATS: INSERT: error")
		return errors.Wrapf(err, "failed to insert statistic %v", row)
	}

	if res.RowsAffected() == 0 {
		panic("=== PANIC === BLOCKCHAIN_STATS: INSERT: rows affected 0")
		s.errorCounter.Inc()
		s.log.WithField("statistic_row", row).
			Errorf("failed to insert statistic")
	}
	return nil
}

func (s *StatisticStorage) Last() *observer.Statistic {
	var err error
	statistic := &StatisticSchema{}

	cycle.UntilError(func() error {
		s.log.Info("trying to get last statistic from db")
		err = s.db.Model(statistic).
			Order("pulse_num DESC").
			Limit(1).
			Select()
		if err != nil && err != pg.ErrNoRows {
			s.log.Error(errors.Wrapf(err, "failed request to db"))
		}
		if err == pg.ErrNoRows {
			return nil
		}
		return err
	}, s.cfg.DB.AttemptInterval, s.cfg.DB.Attempts)

	if err != nil && err != pg.ErrNoRows {
		s.log.Debug("failed to find last pulse row")
		return nil
	}

	if err == pg.ErrNoRows {
		return &observer.Statistic{}
	}

	model := &observer.Statistic{
		Pulse:              statistic.Pulse,
		Transfers:          statistic.Transfers,
		TotalTransfers:     statistic.TotalTransfers,
		TotalMembers:       statistic.TotalMembers,
		MaxTransfers:       statistic.MaxTransfers,
		LastMonthTransfers: statistic.LastMonthTransfers,
		Nodes:              statistic.Nodes,
	}
	return model
}

func statisticSchema(model *observer.Statistic) *StatisticSchema {
	return &StatisticSchema{
		Pulse:              model.Pulse,
		Transfers:          model.Transfers,
		TotalTransfers:     model.TotalTransfers,
		TotalMembers:       model.TotalMembers,
		MaxTransfers:       model.MaxTransfers,
		LastMonthTransfers: model.LastMonthTransfers,
		Nodes:              model.Nodes,
	}
}
