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
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/v2/internal/app/observer"
)

type PulseSchema struct {
	tableName struct{} `sql:"pulses"`

	Pulse     uint32 `sql:",pk"`
	PulseDate int64  `sql:",notnull"`
	Entropy   []byte `sql:",notnull"`
}

type PulseStorage struct {
	log *logrus.Logger
	db  orm.DB
}

func NewPulseStorage(log *logrus.Logger, db orm.DB) *PulseStorage {
	return &PulseStorage{
		log: log,
		db:  db,
	}
}

func (s *PulseStorage) Insert(model *observer.Pulse) error {
	log := s.log
	if model == nil {
		log.Warnf("trying to insert nil pulse model")
		return nil
	}
	row := pulseSchema(model)
	return s.db.Insert(row)
}

func (s *PulseStorage) Last() *observer.Pulse {
	log := s.log
	pulse := &PulseSchema{}
	if err := s.db.Model(pulse).Last(); err != nil {
		log.Debug("failed to find last pulse row")
		return nil
	}
	model := &observer.Pulse{
		Number:    insolar.PulseNumber(pulse.Pulse),
		Timestamp: pulse.PulseDate,
	}
	if err := model.Entropy.Unmarshal(pulse.Entropy); err != nil {
		log.WithField("entropy", pulse.Entropy).
			Debug("failed to unmarshal entropy from db schema to model")
	}
	return model
}

func pulseSchema(model *observer.Pulse) *PulseSchema {
	if model == nil {
		return nil
	}

	return &PulseSchema{
		Pulse:     uint32(model.Number),
		PulseDate: model.Timestamp,
		Entropy:   model.Entropy[:],
	}
}
