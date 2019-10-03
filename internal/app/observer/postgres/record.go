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

type RecordSchema struct {
	tableName struct{} `sql:"records"`

	ID    uint   `sql:",pk_id"`
	Pulse uint32 `sql:",notnull"`
	Key   []byte `sql:",notnull,unique"`
	Value []byte
}

type RecordStorage struct {
	cfg          *configuration.Configuration
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewRecordStorage(cfg *configuration.Configuration, obs *observability.Observability, db orm.DB) *RecordStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_record_storage_error_counter",
		Help: "",
	})
	return &RecordStorage{
		cfg:          cfg,
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *RecordStorage) Insert(model *observer.Record) error {
	if model == nil {
		s.log.Warnf("trying to insert nil record model")
		return nil
	}
	row := recordSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert()

	if err != nil {
		return errors.Wrapf(err, "failed to insert record %v", row)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("record_row", row).
			Errorf("failed to insert record")
	}
	return nil
}

func (s *RecordStorage) Last() *observer.Record {
	var err error
	record := &RecordSchema{}

	cycle.UntilError(func() error {
		s.log.Info("trying to get last record from db")
		err = s.db.Model(record).
			Order("pulse DESC").
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
		s.log.Debug("failed to find last record row")
		return nil
	}

	if err == pg.ErrNoRows {
		return &observer.Record{}
	}

	model := &observer.Record{}
	err = model.Unmarshal(record.Value)
	if err != nil {
		s.log.WithField("record_value", record.Value).
			Debug("failed to unmarshal record.Value from db schema to model")
		return nil
	}
	return model
}

func (s *RecordStorage) Count(by insolar.PulseNumber) uint32 {
	count, err := s.db.Model(&RecordSchema{}).Where("pulse=?", by).Count()
	if err != nil {
		return 0
	}
	return uint32(count)
}

func recordSchema(model *observer.Record) *RecordSchema {
	data, _ := model.Marshal()
	return &RecordSchema{
		Pulse: uint32(model.ID.Pulse()),
		Key:   model.ID.Bytes(),
		Value: data,
	}
}
