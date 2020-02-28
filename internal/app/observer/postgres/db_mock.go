// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres

import (
	"context"

	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
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
	log   insolar.Logger
	model []interface{}
}

func makeResult(log insolar.Logger, model ...interface{}) orm.Result {
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
