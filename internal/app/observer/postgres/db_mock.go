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
	"context"

	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"
)

type DBMock struct {
	orm.DB
	insert   func(model ...interface{}) error
	model    func(model ...interface{}) *orm.Query
	queryOne func(model, query interface{}, params ...interface{}) (orm.Result, error)
	query    func(model, query interface{}, params ...interface{}) (orm.Result, error)
}

func (m *DBMock) Insert(model ...interface{}) error {
	return m.insert(model...)
}

func (m *DBMock) Model(model ...interface{}) *orm.Query {
	return m.model(model...)
}

func (m *DBMock) QueryOne(model, query interface{}, params ...interface{}) (orm.Result, error) {
	return m.queryOne(model, query, params...)
}

func (m *DBMock) Query(model, query interface{}, params ...interface{}) (orm.Result, error) {
	return m.query(model, query, params...)
}

func (m *DBMock) QueryContext(_ context.Context, model, query interface{}, params ...interface{}) (orm.Result, error) {
	return m.Query(model, query, params...)
}

func (m *DBMock) QueryOneContext(_ context.Context, model, query interface{}, params ...interface{}) (orm.Result, error) {
	return m.QueryOne(model, query, params...)
}

type resultMock struct {
	orm.Result
	log   *logrus.Logger
	model []interface{}
}

func makeResult(log *logrus.Logger, model ...interface{}) orm.Result {
	return &resultMock{log: log, model: model}
}

func (m *resultMock) Model() orm.Model {
	model, err := orm.NewModel(m.model...)
	if err != nil {
		m.log.Info(err)
		return nil
	}
	return model
}

func (m *resultMock) RowsReturned() int {
	return len(m.model)
}

func (m *resultMock) RowsAffected() int {
	return len(m.model)
}
