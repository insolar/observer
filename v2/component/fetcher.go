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
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/observer/v2/configuration"
	"github.com/insolar/observer/v2/internal/app/observer/grpc"
	"github.com/insolar/observer/v2/internal/app/observer/postgres"
)

func makeFetcher(cfg *configuration.Configuration, obs *Observability, conn *Connectivity) func() *raw {
	log := obs.Log()
	lastPulse, recordCounter := fetchingMetrics(obs)
	db := conn.PG().DB()
	pulseClient := exporter.NewPulseExporterClient(conn.GRPC().Conn())
	recordClient := exporter.NewRecordExporterClient(conn.GRPC().Conn())
	pulses := grpc.NewPulseFetcher(pulseClient, postgres.NewPulseStorage(log, db))
	records := grpc.NewRecordFetcher(cfg, recordClient, postgres.NewRecordStorage(log, db))
	return func() *raw {
		pulse, err := pulses.Fetch()
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to fetch pulse"))
			return nil
		}
		lastPulse.Set(float64(pulse.Number))
		log.Infof("fetched %d pulse", pulse.Number)

		batch, err := records.Fetch(pulse.Number)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to fetch records by pulse"))
			return nil
		}
		recordCounter.Add(float64(len(batch)))
		log.Infof("fetched %d records", len(batch))
		return &raw{pulse: pulse, batch: batch}
	}
}

func fetchingMetrics(obs *Observability) (prometheus.Gauge, prometheus.Counter) {
	log := obs.Log()
	metrics := obs.Metrics()
	lastPulse := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "observer_last_sync_pulse",
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
