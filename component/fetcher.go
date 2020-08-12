// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package component

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/grpc"
	"github.com/insolar/observer/observability"
)

func makeFetcher(
	obs *observability.Observability,
	pulses observer.PulseFetcher,
	records observer.HeavyRecordFetcher,
) func(context.Context, *state) *raw {
	log := obs.Log()
	lastPulseMetric, recordCounterMetric := fetchingMetrics(obs)

	return func(ctx context.Context, s *state) *raw {
		// Get next pulse
		// todo: get batch of empty pulses, if shouldIterateFrom set
		pulse, err := pulses.Fetch(ctx, s.last)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to fetch pulse"))
			return nil
		}
		lastPulseMetric.Set(float64(pulse.Number))
		log.WithField("number", pulse.Number).
			Debug("fetched pulse record")

		// Early return of empty pulses
		if pulse.Number < s.ShouldIterateFrom {
			log.WithField("should_iterate_from", s.ShouldIterateFrom).
				Debug("skipped record fetching")
			return &raw{pulse: pulse, shouldIterateFrom: s.ShouldIterateFrom, currentHeavyPN: s.ShouldIterateFrom}
		}

		// Get records
		batch, shouldIterateFrom, err := records.Fetch(ctx, pulse.Number)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to fetch records by pulse"))
			return nil
		}
		recordCounterMetric.Add(float64(len(batch)))
		log.WithField("batch_size", len(batch)).
			Infof("fetched records")
		if pulse.Number < s.currentHeavyPN {
			return &raw{pulse: pulse, batch: batch, shouldIterateFrom: shouldIterateFrom, currentHeavyPN: s.currentHeavyPN}
		}
		// Get current heavy pulse
		currentFinalisedPulse, err := pulses.FetchCurrent(ctx)
		if err != nil {
			return &raw{pulse: pulse, batch: batch, shouldIterateFrom: shouldIterateFrom}
		}
		s.currentHeavyPN = currentFinalisedPulse
		return &raw{pulse: pulse, batch: batch, shouldIterateFrom: shouldIterateFrom, currentHeavyPN: currentFinalisedPulse}
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
	grpc.NewDeprecatedClientMetric(obs)
	return lastPulse, recordCounter
}
