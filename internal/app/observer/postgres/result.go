// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres

import (
	"encoding/hex"

	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

type ResultSchema struct {
	tableName struct{} `sql:"results"` //nolint: unused,structcheck

	ResultID string `sql:"result_id,pk"`
	Request  string
	Payload  string
}

type ResultStorage struct {
	log          insolar.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewResultStorage(obs *observability.Observability, db orm.DB) *ResultStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_result_storage_error_counter",
		Help: "",
	})
	return &ResultStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *ResultStorage) Insert(model *observer.Result) error {
	if model == nil {
		s.log.Warnf("trying to insert nil result model")
		return nil
	}
	row := resultSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert()

	if err != nil {
		return errors.Wrapf(err, "failed to insert result %v", row)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("result_row", row).
			Errorf("failed to insert result")
	}
	return nil
}

func resultSchema(model *observer.Result) *ResultSchema {
	res := model.Virtual.GetResult()
	return &ResultSchema{
		ResultID: model.ID.String(),
		Request:  res.Request.String(),
		Payload:  hex.EncodeToString(res.Payload),
	}
}
