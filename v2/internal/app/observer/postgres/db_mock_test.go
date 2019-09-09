package postgres

import (
	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"
)

type dbMock struct {
	orm.DB
	insert   func(model ...interface{}) error
	model    func(model ...interface{}) *orm.Query
	queryOne func(model, query interface{}, params ...interface{}) (orm.Result, error)
}

func (m *dbMock) Insert(model ...interface{}) error {
	return m.insert(model...)
}

func (m *dbMock) Model(model ...interface{}) *orm.Query {
	return m.model(model...)
}

func (m *dbMock) QueryOne(model, query interface{}, params ...interface{}) (orm.Result, error) {
	return m.queryOne(model, query, params...)
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
