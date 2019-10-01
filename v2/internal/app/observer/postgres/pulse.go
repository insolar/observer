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

	"github.com/insolar/observer/v2/configuration"
	"github.com/insolar/observer/v2/internal/app/observer"
	"github.com/insolar/observer/v2/internal/pkg/cycle"
	"github.com/insolar/observer/v2/observability"
)

type PulseSchema struct {
	tableName struct{} `sql:"pulses"`

	Pulse     uint32 `sql:",pk"`
	PulseDate int64  `sql:",notnull"`
	Entropy   []byte `sql:",notnull"`
}

type PulseStorage struct {
	cfg          *configuration.Configuration
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewPulseStorage(cfg *configuration.Configuration, obs *observability.Observability, db orm.DB) *PulseStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_pulse_storage_error_counter",
		Help: "",
	})
	return &PulseStorage{
		cfg:          cfg,
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *PulseStorage) Insert(model *observer.Pulse) error {
	if model == nil {
		s.log.Warnf("trying to insert nil pulse model")
		return nil
	}
	row := pulseSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert(row)

	if err != nil {
		return errors.Wrapf(err, "failed to insert pulse %v", row)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("pulse_row", row).
			Errorf("failed to insert pulse")
	}
	return nil
}

func (s *PulseStorage) Last() *observer.Pulse {
	var err error
	pulse := &PulseSchema{}

	cycle.UntilError(func() error {
		err = s.db.Model(pulse).
			Order("pulse DESC").
			Limit(1).
			Select()
		if err != nil && err != pg.ErrNoRows {
			s.log.Error(errors.Wrapf(err, "failed request to db"))
		}
		return err
	}, s.cfg.DB.AttemptInterval, s.cfg.DB.Attempts)

	if err != nil && err != pg.ErrNoRows {
		s.log.Debug("failed to find last pulse row")
		return nil
	}

	if err == pg.ErrNoRows {
		return &observer.Pulse{}
	}

	model := &observer.Pulse{
		Number:    insolar.PulseNumber(pulse.Pulse),
		Timestamp: pulse.PulseDate,
	}
	if err := model.Entropy.Unmarshal(pulse.Entropy); err != nil {
		s.log.WithField("entropy", pulse.Entropy).
			Debug("failed to unmarshal entropy from db schema to model")
	}
	return model
}

func pulseSchema(model *observer.Pulse) *PulseSchema {
	return &PulseSchema{
		Pulse:     uint32(model.Number),
		PulseDate: model.Timestamp,
		Entropy:   model.Entropy[:],
	}
}
