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
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/connectivity"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/observability"
)

func makeInitter(cfg *configuration.Configuration, obs *observability.Observability, conn *connectivity.Connectivity) func() *state {
	last := MustKnowPulse(cfg, obs, conn.PG())
	DBSchemaMustBeSupported(cfg, obs, conn.PG())
	recordPosition := MustKnowRecordPosition(cfg, obs, conn.PG())
	return func() *state {
		return &state{
			last: last,
			rp:   recordPosition,
		}
	}
}

func DBSchemaMustBeSupported(cfg *configuration.Configuration, obs *observability.Observability, db orm.DB) {
	migrations := postgres.NewChangeLogStorage(cfg, obs, db)
	isRequiredChangeSet := migrations.CheckVersion(cfg.DB.Migration)
	if !isRequiredChangeSet {
		panic("Version of DB schema is not supported please apply corresponding migration")
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
