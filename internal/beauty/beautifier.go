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

package beauty

import (
	"context"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/insolar/observer/internal/beauty/burn"
	"github.com/insolar/observer/internal/beauty/deposit"
	depositTransfer "github.com/insolar/observer/internal/beauty/deposit/transfer"
	"github.com/insolar/observer/internal/beauty/member"
	"github.com/insolar/observer/internal/beauty/member/transfer"
	"github.com/insolar/observer/internal/configuration"
	"github.com/insolar/observer/internal/db"
	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/replication"

	log "github.com/sirupsen/logrus"
)

func NewBeautifier() *Beautifier {
	migrationAddressGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "observer_available_migration_addresses_total",
		Help: "Cache size of migration address composer",
	})
	return &Beautifier{
		cfg:                      configuration.Default(),
		memberComposer:           member.NewComposer(),
		memberBalanceUpdater:     member.NewBalanceUpdater(),
		transferComposer:         transfer.NewComposer(),
		depositComposer:          deposit.NewComposer(),
		migrationAddressComposer: burn.NewComposer(migrationAddressGauge),
		migrationAddressKeeper:   burn.NewKeeper(migrationAddressGauge),
		depositTransferComposer:  depositTransfer.NewComposer(),
		depositKeeper:            deposit.NewKeeper(),
	}
}

type Beautifier struct {
	Configurator     configuration.Configurator `inject:""`
	OnData           replication.OnData         `inject:""`
	OnDump           replication.OnDump         `inject:""`
	ConnectionHolder db.ConnectionHolder        `inject:""`
	cfg              *configuration.Configuration

	memberComposer           *member.Composer
	memberBalanceUpdater     *member.BalanceUpdater
	transferComposer         *transfer.Composer
	depositComposer          *deposit.Composer
	migrationAddressComposer *burn.Composer
	migrationAddressKeeper   *burn.MigrationAddressKeeper
	depositTransferComposer  *depositTransfer.Composer
	depositKeeper            *deposit.DepositKeeper
}

type Record struct {
	tableName struct{} `sql:"records"`

	Key   string
	Value string
	Scope uint
}

// Init initializes connection to db and subscribes beautifier on db updates.
func (b *Beautifier) Init(ctx context.Context) error {
	if b.Configurator != nil {
		b.cfg = b.Configurator.Actual()
	} else {
		b.cfg = configuration.Default()
	}
	if b.OnData != nil {
		b.OnData.SubscribeOnData(func(recordNumber uint32, rec *record.Material) {
			b.process(rec)
		})
	}
	if b.OnDump != nil {
		b.OnDump.SubscribeOnDump(b.dump)
	}
	if b.cfg.DB.CreateTables {
		b.createTables()
	}
	if b.ConnectionHolder != nil {
		b.migrationAddressComposer.Init(b.ConnectionHolder.DB())
	}
	return nil
}

func (b *Beautifier) createTables() {
	if b.ConnectionHolder != nil {
		db := b.ConnectionHolder.DB()
		if err := db.CreateTable(&beauty.Transfer{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
			log.Error(errors.Wrapf(err, "failed to create transactions table"))
		}
		if err := db.CreateTable(&beauty.Member{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
			log.Error(errors.Wrapf(err, "failed to create members table"))
		}
		if err := db.CreateTable(&beauty.Deposit{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
			log.Error(errors.Wrapf(err, "failed to create deposits table"))
		}
		if err := db.CreateTable(&beauty.MigrationAddress{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
			log.Error(errors.Wrapf(err, "failed to create migrations_addresses table"))
		}
	}
}

func (b *Beautifier) process(rec *record.Material) {
	b.memberComposer.Process(rec)
	b.memberBalanceUpdater.Process(rec)
	b.transferComposer.Process(rec)
	b.depositComposer.Process(rec)
	b.migrationAddressComposer.Process(rec)
	b.migrationAddressKeeper.Process(rec)
	b.depositTransferComposer.Process(rec)
	b.depositKeeper.Process(rec)
}

func (b *Beautifier) dump(tx *pg.Tx, pub replication.OnDumpSuccess) error {
	if err := b.memberComposer.Dump(tx, pub); err != nil {
		return err
	}
	if err := b.memberBalanceUpdater.Dump(tx, pub); err != nil {
		return err
	}
	if err := b.transferComposer.Dump(tx, pub); err != nil {
		return err
	}
	if err := b.depositComposer.Dump(tx, pub); err != nil {
		return err
	}
	if err := b.migrationAddressComposer.Dump(tx, pub); err != nil {
		return err
	}
	if err := b.migrationAddressKeeper.Dump(tx, pub); err != nil {
		return err
	}
	if err := b.depositTransferComposer.Dump(tx, pub); err != nil {
		return err
	}
	if err := b.depositKeeper.Dump(tx, pub); err != nil {
		return err
	}
	return nil
}
