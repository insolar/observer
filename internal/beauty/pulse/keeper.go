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

package pulse

import (
	"context"
	"encoding/hex"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/configuration"
	"github.com/insolar/observer/internal/db"
	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/replication"
)

type Keeper struct {
	Configurator     configuration.Configurator `inject:""`
	OnPulse          replication.OnPulse        `inject:""`
	OnDump           replication.OnDump         `inject:""`
	ConnectionHolder db.ConnectionHolder        `inject:""`
	cfg              *configuration.Configuration

	cache []*beauty.Pulse
}

func NewKeeper() *Keeper {
	return &Keeper{}
}

func (k *Keeper) Init(ctx context.Context) error {
	if k.Configurator != nil {
		k.cfg = k.Configurator.Actual()
	} else {
		k.cfg = configuration.Default()
	}
	if k.OnPulse != nil {
		k.OnPulse.SubscribeOnPulse(k.process)
	}
	if k.OnDump != nil {
		k.OnDump.SubscribeOnDump(k.dump)
	}
	if k.cfg.DB.CreateTables {
		k.createTables()
	}
	return nil
}

func (k *Keeper) createTables() {
	if k.ConnectionHolder != nil {
		db := k.ConnectionHolder.DB()
		if err := db.CreateTable(&beauty.Pulse{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
			log.Error(errors.Wrapf(err, "failed to create transactions table"))
		}
	}
}

func (k *Keeper) process(pn insolar.PulseNumber, entropy insolar.Entropy, timestamp int64) {
	k.cache = append(k.cache, &beauty.Pulse{
		Pulse:     pn,
		PulseDate: timestamp,
		Entropy:   hex.EncodeToString(entropy[:]),
	})
}

func (k *Keeper) dump(tx *pg.Tx, pub replication.OnDumpSuccess) error {
	for _, p := range k.cache {
		if err := p.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump pulse")
		}
	}

	pub.Subscribe(func() {
		k.cache = []*beauty.Pulse{}
	})
	return nil
}
