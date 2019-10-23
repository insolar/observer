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
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

func makeFetcher(
	obs *observability.Observability,
	pulses observer.PulseFetcher,
	records observer.RecordFetcher,
) func(*state) *raw {
	log := obs.Log()
	lastPulseMetric, recordCounterMetric := fetchingMetrics(obs)

	return func(s *state) *raw {
		// Get next pulse
		// todo: get batch of empty pulses, if shouldIterateFrom set
		pulse, err := pulses.Fetch(s.last)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to fetch pulse"))
			return nil
		}
		lastPulseMetric.Set(float64(pulse.Number))
		log.WithField("number", pulse.Number).
			Debug("fetched pulse record")

		// Early return of empty pulses
		if pulse.Number < s.rp.ShouldIterateFrom {
			log.WithField("should_iterate_from", s.rp.ShouldIterateFrom).
				Debug("skipped record fetching")
			return &raw{pulse: pulse, shouldIterateFrom: s.rp.ShouldIterateFrom}
		}

		// Get records
		batch, shouldIterateFrom, err := records.Fetch(pulse.Number)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to fetch records by pulse"))
			return nil
		}
		recordCounterMetric.Add(float64(len(batch)))
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
