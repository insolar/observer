package postgres

import (
	"time"

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

func (s *PulseStorage) GetRange(fromTimestamp, toTimestamp int64, limit int, pulseNumber *int64) ([]uint32, error) {
	var err error
	var pulses []models.Pulse

	s.log.Info("trying to get range of pulses from db")
	query := s.db.Model(&pulses)
	if pulseNumber != nil {
		query = query.Where("pulse > ?", uint32(*pulseNumber))
	}
	// we store pulse date as UnixNano, so we multiply timestamp to seconds
	err = query.
		Order("pulse_date ASC").
		Where("pulse_date >= ?", fromTimestamp*time.Second.Nanoseconds()).
		Where("pulse_date <= ?", toTimestamp*time.Second.Nanoseconds()).
		Limit(limit).
		Select()
	if err == pg.ErrNoRows {
		s.log.Warnf("no pulses in timestamp range %d - %d in db", fromTimestamp, toTimestamp)
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed request to db")
	}

	var res []uint32
	for _, p := range pulses {
		res = append(res, p.Pulse)
	}

	return res, nil
}

func pulseSchema(model *observer.Pulse) *models.Pulse {
	return &models.Pulse{
		Pulse:     uint32(model.Number),
		PulseDate: model.Timestamp,
		Entropy:   model.Entropy[:],
		Nodes:     uint32(len(model.Nodes)),
	}
}
