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
	"strings"
	"time"

	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type MigrationResult string

const (
	MigrationResultOK            MigrationResult = "ok"
	MigrationResultExpectedError                 = "expected_error"
	MigrationResultError                         = "error"
)

type MigrationStatsModel struct {
	tableName struct{} `sql:"xns_migration_stats"` //nolint: unused,structcheck

	ID                  uint64          `sql:"id,pk"`
	DaemonID            string          `sql:"daemon_id"`
	InsolarRef          []byte          `sql:"insolar_ref"`
	ModificationTime    time.Time       `sql:"modification_time default:now()"`
	EthBlock            uint64          `sql:"eth_block"`
	TxHash              string          `sql:"tx_hash"`
	Amount              uint64          `sql:"amount"`
	Result              MigrationResult `sql:"name:result type:xns_migrations_status"`
	ContractRequestBody *string         `sql:"contract_request_body"`
	Error               *string         `sql:"error"`
}

//go:generate minimock -i MigrationStatsRepo -o ./ -s _mock.go -g

type MigrationStatsRepo interface {
	Insert(model *MigrationStatsModel) error
}

type MigrationStatsRepository struct {
	db  orm.DB
	log *logrus.Logger
}

func NewMigrationStatsRepository(db orm.DB, log *logrus.Logger) *MigrationStatsRepository {
	return &MigrationStatsRepository{db: db, log: log}
}

var DuplicatedMigration = errors.New("it's impossible to save duplicate of the migration")

func (m *MigrationStatsRepository) Insert(model *MigrationStatsModel) error {
	if model == nil {
		err := errors.New("trying to insert nil migration stats model")
		m.log.Error(err)
		return err
	}
	model.ModificationTime = time.Now()
	err := m.db.Insert(model)
	if err != nil {
		if strings.Contains(err.Error(), `duplicate key value violates unique constraint "xns_migration_stats_daemon_id_eth_block_tx_hash_amount_key"`) {
			return DuplicatedMigration
		}
		return errors.Wrap(err, "failed to insert stats")
	}

	return nil
}
