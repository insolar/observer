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
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

type MigrationAddressSchema struct {
	tableName struct{} `sql:"migration_addresses"`

	Addr      string `sql:",pk"`
	Timestamp int64  `sql:",notnull"`
	Wasted    bool
}

type MigrationAddressStorage struct {
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewMigrationAddressStorage(obs *observability.Observability, db orm.DB) *MigrationAddressStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_migration_address_storage_error_counter",
		Help: "",
	})
	return &MigrationAddressStorage{
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
	}
	return nil
}

func (s *MigrationAddressStorage) Update(model *observer.Wasting) error {
	if model == nil {
		s.log.Warnf("trying to apply nil wasting model")
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
	}
	return nil
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
