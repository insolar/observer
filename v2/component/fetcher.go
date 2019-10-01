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
	"github.com/insolar/observer/v2/connectivity"
	"github.com/insolar/observer/v2/internal/app/observer/grpc"
	"github.com/insolar/observer/v2/observability"
)

func makeFetcher(
	cfg *configuration.Configuration,
	obs *observability.Observability,
	conn *connectivity.Connectivity,
) func(*state) *raw {
	log := obs.Log()
	lastPulse, recordCounter := fetchingMetrics(obs)
	pulseClient := exporter.NewPulseExporterClient(conn.GRPC())
	recordClient := exporter.NewRecordExporterClient(conn.GRPC())
	pulses := grpc.NewPulseFetcher(pulseClient)
	records := grpc.NewRecordFetcher(cfg, obs, recordClient)
	return func(s *state) *raw {
		pulse, err := pulses.Fetch(s.last)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to fetch pulse"))
			return nil
		}
		lastPulse.Set(float64(pulse.Number))
		log.WithField("number", pulse.Number).
			Infof("fetched pulse record")

		if pulse.Number < s.rp.ShouldIterateFrom {
			log.WithField("should_iterate_from", s.rp.ShouldIterateFrom).
				Infof("skipped record fetching")
			return &raw{pulse: pulse, shouldIterateFrom: s.rp.ShouldIterateFrom}
		}

		batch, shouldIterateFrom, err := records.Fetch(pulse.Number)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to fetch records by pulse"))
			return nil
		}
		recordCounter.Add(float64(len(batch)))
		log.WithField("batch_size", len(batch)).
			Infof("fetched records")
		return &raw{pulse: pulse, batch: batch, shouldIterateFrom: shouldIterateFrom}
	}
}

func fetchingMetrics(obs *observability.Observability) (prometheus.Gauge, prometheus.Counter) {
	lastPulse := obs.Gauge(prometheus.GaugeOpts{
		Name: "observer_last_fetched_pulse",
		Help: "Last pulse that was fetched from HME.",
	})
	recordCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_fetched_record_total",
		Help: "Number of records fetched from HME.",
	})
	return lastPulse, recordCounter
}
