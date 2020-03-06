// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres

import (
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/models"
)

type PulseStorage struct {
	log insolar.Logger
	db  orm.DB
}

func NewPulseStorage(log insolar.Logger, db orm.DB) *PulseStorage {
	return &PulseStorage{
		log: log,
		db:  db,
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
		Insert()

	if err != nil {
		return errors.Wrapf(err, "failed to insert pulse %v", row)
	}

	if res.RowsAffected() == 0 {
		s.log.WithField("pulse_row", row).
			Errorf("failed to insert pulse")
		return nil
	}
	return nil
}

func (s *PulseStorage) Last() (*observer.Pulse, error) {
	var err error
	pulse := &models.Pulse{}

	s.log.Info("trying to get last pulse from db")
	err = s.db.Model(pulse).
		Order("pulse DESC").
		Limit(1).
		Select()
	if err == pg.ErrNoRows {
		s.log.Warn("no pulses in db")
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed request to db")
	}

	model := &observer.Pulse{
		Number:    insolar.PulseNumber(pulse.Pulse),
		Timestamp: pulse.PulseDate,
	}

	err = model.Entropy.Unmarshal(pulse.Entropy)
	if err != nil {
		s.log.WithField("entropy", pulse.Entropy).
			Error("failed to unmarshal entropy from db schema to model")
	}

	return model, nil
}

func pulseSchema(model *observer.Pulse) *models.Pulse {
	return &models.Pulse{
		Pulse:     uint32(model.Number),
		PulseDate: model.Timestamp,
		Entropy:   model.Entropy[:],
		Nodes:     uint32(len(model.Nodes)),
	}
}
