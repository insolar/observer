// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres

import (
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/pkg/cycle"
	"github.com/insolar/observer/observability"
)

type MigrationAddressSchema struct {
	tableName struct{} `sql:"migration_addresses"` //nolint: unused,structcheck

	Addr      string `sql:",pk"`
	Timestamp int64  `sql:",notnull"`
	Wasted    bool   `sql:"wasted,notnull"`
}

type MigrationAddressStorage struct {
	cfg          *configuration.DB
	log          insolar.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewMigrationAddressStorage(cfg *configuration.DB, obs *observability.Observability, db orm.DB) *MigrationAddressStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_migration_address_storage_error_counter",
		Help: "",
	})
	return &MigrationAddressStorage{
		cfg:          cfg,
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *MigrationAddressStorage) Insert(model *observer.MigrationAddress) error {
	if model == nil {
		s.log.Warnf("trying to insert nil migration_address model")
		return nil
	}
	row := migrationAddressSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert()

	if err != nil {
		return errors.Wrapf(err, "failed to insert migration_address %v", row)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("migration_address_row", row).Errorf("failed to insert migration_address")
		return errors.New("failed to insert, affected is 0")
	}
	return nil
}

func (s *MigrationAddressStorage) Update(model *observer.Wasting) error {
	if model == nil {
		s.log.Warnf("trying to apply nil vasting model")
		return nil
	}

	res, err := s.db.Model(&MigrationAddressSchema{}).
		Where("addr=?", model.Addr).
		Set("wasted=true").
		Update()

	if err != nil {
		return errors.Wrapf(err, "failed to update migration_address upd=%v", model)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("upd", model).Errorf("failed to update migration_address")
		return errors.New("failed to update, affected is 0")
	}
	return nil
}

func (s *MigrationAddressStorage) Wasted() int {
	var err error
	cnt := 0

	cycle.UntilError(func() error {
		s.log.Info("trying to get wasted addresses from db")
		cnt, err = s.db.Model(&MigrationAddressSchema{}).
			Where("wasted is true").
			Count()
		if err != nil && err != pg.ErrNoRows {
			s.log.Error(errors.Wrapf(err, "failed request to db"))
		}
		if err == pg.ErrNoRows {
			return nil
		}
		return err
	}, s.cfg.AttemptInterval, s.cfg.Attempts)

	if err != nil && err != pg.ErrNoRows {
		s.log.Debug("failed to find wasted addresses from db")
		return 0
	}

	if err == pg.ErrNoRows {
		return 0
	}

	return cnt
}

func (s *MigrationAddressStorage) TotalMigrationAddresses() int {
	var err error
	cnt := 0

	cycle.UntilError(func() error {
		s.log.Info("trying to get active addresses from db")
		cnt, err = s.db.Model(&MigrationAddressSchema{}).
			Count()
		if err != nil && err != pg.ErrNoRows {
			s.log.Error(errors.Wrapf(err, "failed request to db"))
		}
		if err == pg.ErrNoRows {
			return nil
		}
		return err
	}, s.cfg.AttemptInterval, s.cfg.Attempts)

	if err != nil && err != pg.ErrNoRows {
		s.log.Debug("failed to find active addresses from db")
		return 0
	}

	if err == pg.ErrNoRows {
		return 0
	}

	return cnt
}

func migrationAddressSchema(model *observer.MigrationAddress) *MigrationAddressSchema {
	t, err := model.Pulse.AsApproximateTime()
	timestamp := int64(0)
	if err == nil {
		timestamp = t.Unix()
	}
	return &MigrationAddressSchema{
		Addr:      model.Addr,
		Timestamp: timestamp,
		Wasted:    false,
	}
}
