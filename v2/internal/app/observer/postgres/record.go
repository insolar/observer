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

type RecordSchema struct {
	tableName struct{} `sql:"records"`

	ID    uint   `sql:",pk_id"`
	Pulse uint32 `sql:",notnull"`
	Key   []byte `sql:",notnull,unique"`
	Value []byte
}

type RecordStorage struct {
	log *logrus.Logger
	db  orm.DB
}

func NewRecordStorage(log *logrus.Logger, db orm.DB) *RecordStorage {
	return &RecordStorage{
		log: log,
		db:  db,
	}
}

func (s *RecordStorage) Insert(model *observer.Record) error {
	row := recordSchema(model)
	if row != nil {
		return s.db.Insert(row)
	}
	return nil
}

func (s *RecordStorage) Last() *observer.Record {
	log := s.log
	record := &RecordSchema{}
	if err := s.db.Model(record).Last(); err != nil {
		log.Debug("failed to find last records row")
		return nil
	}
	model := &observer.Record{}
	err := model.Unmarshal(record.Value)
	if err != nil {
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
	if model == nil {
		return nil
	}

	data, err := model.Marshal()
	if err != nil {
		return nil
	}
	return &RecordSchema{
		Pulse: uint32(model.ID.Pulse()),
		Key:   model.ID.Bytes(),
		Value: data,
	}
}
