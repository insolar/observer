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

package dump

import (
	"context"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/configuration"
	"github.com/insolar/observer/internal/raw"
	"github.com/insolar/observer/internal/replica"
)

type Dumper struct {
	Configurator configuration.Configurator `inject:""`
	Publisher    replica.Publisher          `inject:""`
	cfg          *configuration.Configuration
	db           *pg.DB
	prevPulse    insolar.PulseNumber
	cache        []*raw.Record
}

func NewLoader() *Dumper {
	return &Dumper{}
}

func (d *Dumper) Init(ctx context.Context) error {
	if d.Configurator != nil {
		d.cfg = d.Configurator.Actual()
	} else {
		d.cfg = configuration.Default()
	}
	if d.Publisher != nil {
		d.Publisher.Subscribe(func(recordNumber uint32, rec *record.Material) {
			pn := rec.ID.Pulse()
			d.save(recordNumber, pn, rec)
		})
	}
	opt, err := pg.ParseURL(d.cfg.DB.URL)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to parse cfg.DB.URL"))
		return nil
	}
	d.db = pg.Connect(opt)

	if d.cfg.DB.CreateTables {
		d.createTable()
	}
	return nil
}

func (d *Dumper) createTable() {
	if err := d.db.CreateTable(&raw.Record{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
		log.Error(errors.Wrapf(err, "failed to create records table"))
	}
}

func (d *Dumper) save(recordNumber uint32, pn insolar.PulseNumber, rec *record.Material) {
	if d.prevPulse == 0 {
		d.prevPulse = pn
	}
	if d.prevPulse != pn {
		d.flush(d.prevPulse)
		d.prevPulse = pn
	}

	val, err := rec.Marshal()
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to marshal record value rec=%v", rec))
		return
	}
	d.cache = append(d.cache, &raw.Record{Number: recordNumber, Key: rec.ID.Bytes(), Value: val})
}

func (d *Dumper) flush(pn insolar.PulseNumber) {
	log.WithField("pulse", pn).Debugf("flushing raw records")
	tx, err := d.db.Begin()
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to create db transaction"))
		return
	}
	defer func() {
		err := tx.Commit()
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to commit db transaction"))
		}
	}()
	for _, rec := range d.cache {
		err = tx.Insert(rec)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to insert record into db"))
			return
		}
	}
	d.cache = []*raw.Record{}
}
