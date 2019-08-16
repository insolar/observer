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

package dto

import (
	"context"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/configuration"
	"github.com/insolar/observer/internal/db"
	"github.com/insolar/observer/internal/model/raw"
	"github.com/insolar/observer/internal/replication"
)

type Dumper struct {
	Configurator     configuration.Configurator `inject:""`
	OnData           replication.OnData         `inject:""`
	OnDump           replication.OnDump         `inject:""`
	ConnectionHolder db.ConnectionHolder        `inject:""`
	cfg              *configuration.Configuration
	records          []*raw.Record
	requests         []*raw.Request
	results          []*raw.Result
	objects          []*raw.Object
}

func NewDumper() *Dumper {
	return &Dumper{}
}

func (d *Dumper) Init(ctx context.Context) error {
	if d.Configurator != nil {
		d.cfg = d.Configurator.Actual()
	} else {
		d.cfg = configuration.Default()
	}
	if d.OnData != nil {
		d.OnData.SubscribeOnData(func(rn uint32, rec *record.Material) {
			d.process(rn, rec)
		})
	}
	if d.OnDump != nil {
		d.OnDump.SubscribeOnDump(d.dump)
	}
	if d.cfg.DB.CreateTables {
		d.createTables()
	}
	return nil
}

func (d *Dumper) createTables() {
	if d.ConnectionHolder != nil {
		db := d.ConnectionHolder.DB()
		if err := db.CreateTable(&raw.Record{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
			log.Error(errors.Wrapf(err, "failed to create records table"))
		}
		if err := db.CreateTable(&raw.Object{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
			log.Error(errors.Wrapf(err, "failed to create objects table"))
		}
		if err := db.CreateTable(&raw.Request{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
			log.Error(errors.Wrapf(err, "failed to create requests table"))
		}
		if err := db.CreateTable(&raw.Result{}, &orm.CreateTableOptions{IfNotExists: true}); err != nil {
			log.Error(errors.Wrapf(err, "failed to create results table"))
		}
	}
}

func (d *Dumper) process(rn uint32, rec *record.Material) {
	d.buildRecord(rn, rec)
	d.buildUnpacked(rec)
}

func (d *Dumper) dump(tx *pg.Tx, pub replication.OnDumpSuccess) error {
	log.Infof("dump raw records")
	for _, rec := range d.records {
		if err := rec.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump raw records")
		}
	}
	for _, req := range d.requests {
		if err := req.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump raw requests")
		}
	}
	for _, res := range d.results {
		if err := res.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump raw results")
		}
	}
	for _, obj := range d.objects {
		if err := obj.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump raw objects")
		}
	}

	pub.Subscribe(func() {
		d.records = []*raw.Record{}
		d.requests = []*raw.Request{}
		d.results = []*raw.Result{}
		d.objects = []*raw.Object{}
	})
	return nil
}

func (d *Dumper) buildRecord(rn uint32, rec *record.Material) {
	val, err := rec.Marshal()
	if err != nil {
		log.Error(errors.New("failed to marshal raw record"))
		return
	}
	d.records = append(d.records, &raw.Record{Number: rn, Key: rec.ID.Bytes(), Value: val})
}

func (d *Dumper) buildUnpacked(rec *record.Material) {
	switch rec.Virtual.Union.(type) {
	case *record.Virtual_Result:
		d.results = append(d.results, (*Result)(rec).MapModel())
	case *record.Virtual_IncomingRequest:
		d.requests = append(d.requests, parseRequest(rec))
	case *record.Virtual_Activate:
		d.objects = append(d.objects, parseActivate(rec))
	case *record.Virtual_Amend:
		d.objects = append(d.objects, parseAmend(rec))
	case *record.Virtual_Deactivate:
		d.objects = append(d.objects, parseDeactivate(rec))
	}
}
