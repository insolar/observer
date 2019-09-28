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
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/observer/v2/configuration"
	"github.com/insolar/observer/v2/connectivity"
	"github.com/insolar/observer/v2/internal/app/observer/grpc"
	"github.com/insolar/observer/v2/internal/app/observer/postgres"
	"github.com/insolar/observer/v2/observability"
)

func makeFetcher(cfg *configuration.Configuration, obs *observability.Observability, conn *connectivity.Connectivity) func() *raw {
	log := obs.Log()
	db := conn.PG()
	lastPulse, recordCounter := fetchingMetrics(obs)
	pulseClient := exporter.NewPulseExporterClient(conn.GRPC())
	recordClient := exporter.NewRecordExporterClient(conn.GRPC())
	pulses := grpc.NewPulseFetcher(pulseClient)
	records := grpc.NewRecordFetcher(cfg, recordClient, postgres.NewRecordStorage(obs, db))
	last := MustKnowPulse(cfg, obs, db)
	return func() *raw {
		pulse, err := pulses.Fetch(last)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to fetch pulse"))
			return nil
		}
		lastPulse.Set(float64(pulse.Number))
		log.WithField("number", pulse.Number).
			Infof("fetched pulse record")

		batch, err := records.Fetch(pulse.Number)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to fetch records by pulse"))
			return nil
		}
		recordCounter.Add(float64(len(batch)))
		log.WithField("batch_size", len(batch)).
			Infof("fetched records")

		last = pulse.Number
		return &raw{pulse: pulse, batch: batch}
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

func fetchingMetrics(obs *observability.Observability) (prometheus.Gauge, prometheus.Counter) {
	log := obs.Log()
	metrics := obs.Metrics()
	lastPulse := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "observer_last_fetched_pulse",
		Help: "Last pulse that was fetched from HME.",
	})
	recordCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "observer_fetched_record_total",
		Help: "Number of records fetched from HME.",
	})
	err := metrics.Register(lastPulse)
	if err != nil {
		log.Errorf("failed to register last pulse gauge")
	}
	err = metrics.Register(recordCounter)
	if err != nil {
		log.Errorf("failed to register record counter")
	}
	return lastPulse, recordCounter
}
