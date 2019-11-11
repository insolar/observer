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
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/observability"
)

func makeInitter(cfg *configuration.Configuration, obs *observability.Observability, conn PGer) func() *state {
	logger := obs.Log()
	last := MustKnowPulse(cfg, obs, conn.PG())
	recordPosition := MustKnowRecordPosition(cfg, obs, conn.PG())
	metricState := getMetricState(cfg, obs, conn.PG())
	st := state{
		last: last,
		rp:   recordPosition,
		ms:   metricState,
	}
	logger.Debugf("State restored: %+v", st)
	logger.Debugf("Metric state restored : %+v", metricState)
	return func() *state {
		return &st
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

type metricState struct {
	totalWasting            int
	totalMigrationAddresses int
}

func (ms *metricState) Reset() {
	ms.totalWasting = 0
	ms.totalMigrationAddresses = 0
}

func getMetricState(cfg *configuration.Configuration, obs *observability.Observability, db orm.DB) metricState {
	ma := postgres.NewMigrationAddressStorage(cfg, obs, db)
	return metricState{
		totalWasting:            ma.Wasted(),
		totalMigrationAddresses: ma.TotalMigrationAddresses(),
	}
}
