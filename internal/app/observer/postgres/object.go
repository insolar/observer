// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres

import (
	"encoding/hex"

	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

type ObjectSchema struct {
	tableName struct{} `sql:"objects"` //nolint: unused,structcheck

	ObjectID  string `sql:"object_id,pk"`
	Domain    string
	Request   string
	Memory    string
	Image     string
	Parent    string
	PrevState string
	Type      string
}

type ObjectStorage struct {
	log          insolar.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewObjectStorage(obs *observability.Observability, db orm.DB) *ObjectStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_object_storage_error_counter",
		Help: "",
	})
	return &ObjectStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *ObjectStorage) Insert(model interface{}) error {
	if model == nil {
		s.log.Warnf("trying to insert nil object model")
		return nil
	}

	if !isObject(model) {
		s.log.Warnf("trying to insert non-object model")
		return nil
	}

	row := objectSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert()

	if err != nil {
		return errors.Wrapf(err, "failed to insert object %v", row)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("object_row", row).
			Errorf("failed to insert object")
		return errors.New("failed to insert, affected is 0")
	}
	return nil
}

func isObject(model interface{}) bool {
	switch model.(type) {
	case *observer.Activate, *observer.Amend, *observer.Deactivate:
		return true
	}
	return false
}

func objectSchema(model interface{}) *ObjectSchema {
	switch v := model.(type) {
	case *observer.Activate:
		return schemaActivate(v)
	case *observer.Amend:
		return schemaAmend(v)
	case *observer.Deactivate:
		return schemaDeactivate(v)
	}
	return nil
}

func schemaActivate(rec *observer.Activate) *ObjectSchema {
	id := rec.ID
	act := rec.Virtual.GetActivate()
	return &ObjectSchema{
		ObjectID: insolar.NewReference(id).String(),
		Request:  act.Request.String(),
		Memory:   hex.EncodeToString(act.Memory),
		Image:    act.Image.String(),
		Parent:   act.Parent.String(),
		Type:     "ACTIVATE",
	}
}

func schemaAmend(rec *observer.Amend) *ObjectSchema {
	id := rec.ID
	amend := rec.Virtual.GetAmend()
	return &ObjectSchema{
		ObjectID:  insolar.NewReference(id).String(),
		Request:   amend.Request.String(),
		Memory:    hex.EncodeToString(amend.Memory),
		Image:     amend.Image.String(),
		PrevState: amend.PrevState.String(),
		Type:      "AMEND",
	}
}

func schemaDeactivate(rec *observer.Deactivate) *ObjectSchema {
	id := rec.ID
	deact := rec.Virtual.GetDeactivate()
	return &ObjectSchema{
		ObjectID:  insolar.NewReference(id).String(),
		Request:   deact.Request.String(),
		PrevState: deact.PrevState.String(),
		Type:      "DEACTIVATE",
	}
}
