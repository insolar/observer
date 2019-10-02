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

package component

import (
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"

	"github.com/insolar/observer/v2/configuration"
	"github.com/insolar/observer/v2/connectivity"
	"github.com/insolar/observer/v2/internal/app/observer/postgres"
	"github.com/insolar/observer/v2/observability"
)

func makeInitter(cfg *configuration.Configuration, obs *observability.Observability, conn *connectivity.Connectivity) func() *state {
	createTables(cfg, obs, conn)
	initCache()
	last := MustKnowPulse(cfg, obs, conn.PG())
	recordPosition := MustKnowRecordPosition(cfg, obs, conn.PG())
	return func() *state {
		return &state{
			last: last,
			rp:   recordPosition,
		}
	}
}

func MustKnowPulse(cfg *configuration.Configuration, obs *observability.Observability, db orm.DB) insolar.PulseNumber {
	pulses := postgres.NewPulseStorage(cfg, obs, db)
	p := pulses.Last()
	if p == nil {
		panic("Something wrong with DB. Most likely failed to connect to the DB" +
			" in the allotted number of attempts.")
	}
	return p.Number
}

func MustKnowRecordPosition(cfg *configuration.Configuration, obs *observability.Observability, db orm.DB) RecordPosition {
	records := postgres.NewRecordStorage(cfg, obs, db)
	rec := records.Last()
	if rec == nil {
		panic("Something wrong with DB. Most likely failed to connect to the DB" +
			" in the allotted number of attempts.")
	}
	pulse := rec.ID.Pulse()
	rn := records.Count(pulse)
	return RecordPosition{Last: pulse, RN: rn}
}

func createTables(cfg *configuration.Configuration, obs *observability.Observability, conn *connectivity.Connectivity) {
	log := obs.Log()
	if cfg == nil {
		return
	}
	if cfg.DB.CreateTables {
		db := conn.PG()

		err := db.CreateTable(&postgres.PulseSchema{}, &orm.CreateTableOptions{IfNotExists: true})
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to create transfers table"))
		}

		err = db.CreateTable(&postgres.RecordSchema{}, &orm.CreateTableOptions{IfNotExists: true})
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to create transfers table"))
		}

		err = db.CreateTable(&postgres.TransferSchema{}, &orm.CreateTableOptions{IfNotExists: true})
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to create transfers table"))
		}
	}
}

func initCache() {
	// TODO:
	//  1. If dump file exists:
	//  - Load cache from dump file
	//  - Remove file
	//  2. Else if dump record exists in DB:
	//  - Load cache from DB
	//  - Delete record
	//  3. Else if DB connection is alive:
	//  - Init empty cache
	//  4. Else:
	//  - Panic
}
