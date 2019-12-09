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
	"context"
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

func Test_makeFetcher(t *testing.T) {
	mc := minimock.NewController(t)
	ctx := context.Background()
	var obs *observability.Observability
	var pulseFetcher *observer.PulseFetcherMock
	var recordFetcher *observer.HeavyRecordFetcherMock

	resetComponents := func() {
		obs = observability.Make(ctx)
		pulseFetcher = observer.NewPulseFetcherMock(mc)
		recordFetcher = observer.NewHeavyRecordFetcherMock(mc)
	}

	t.Run("happy path", func(t *testing.T) {
		resetComponents()
		defer mc.Finish()

		pn := gen.PulseNumber()
		pulseFetcher.FetchMock.Inspect(func(ctx context.Context, p1 insolar.PulseNumber) {
			assert.Equal(t, pn-10, p1)
		}).Return(&observer.Pulse{
			Number: pn,
		}, nil)

		rec := map[uint32]*exporter.Record{
			0: {Record: record.Material{
				Polymorph: 0,
				Virtual:   record.Virtual{},
				ID:        insolar.ID{},
				ObjectID:  insolar.ID{},
				JetID:     insolar.JetID{},
				Signature: nil,
			},
			},
		}
		recordFetcher.FetchMock.Inspect(func(ctx context.Context, pulse insolar.PulseNumber) {
			assert.Equal(t, pn, pulse)
		}).Return(rec, gen.PulseNumber(), nil)

		pulseFetcher.FetchCurrentMock.Return(pn, nil)
		s := state{
			last: pn - 10,
		}
		fetcher := makeFetcher(obs, pulseFetcher, recordFetcher)
		raw := fetcher(ctx, &s)

		assert.Equal(t, pn, raw.pulse.Number)
		assert.Equal(t, rec, raw.batch)
	})

	t.Run("ShouldIterateFrom early return", func(t *testing.T) {
		resetComponents()
		defer mc.Finish()

		pn := gen.PulseNumber()
		nextActivePulse := pn + 100
		pulseFetcher.FetchMock.Inspect(func(ctx context.Context, p1 insolar.PulseNumber) {
			assert.Equal(t, pn-10, p1)
		}).Return(&observer.Pulse{
			Number: pn,
		}, nil)

		s := state{
			last:              pn - 10,
			ShouldIterateFrom: nextActivePulse,
		}
		fetcher := makeFetcher(obs, pulseFetcher, recordFetcher)
		raw := fetcher(ctx, &s)

		assert.Equal(t, pn, raw.pulse.Number)
		assert.Equal(t, nextActivePulse, raw.shouldIterateFrom)
	})

	t.Run("Failed fetch pulse", func(t *testing.T) {
		resetComponents()
		defer mc.Finish()

		pn := gen.PulseNumber()
		pulseFetcher.FetchMock.Inspect(func(ctx context.Context, p1 insolar.PulseNumber) {
			assert.Equal(t, pn-10, p1)
		}).Return(nil, errors.New("test"))

		s := state{
			last: pn - 10,
		}
		fetcher := makeFetcher(obs, pulseFetcher, recordFetcher)
		raw := fetcher(ctx, &s)
		assert.Nil(t, raw)
	})

	t.Run("Failed fetch records", func(t *testing.T) {
		resetComponents()
		defer mc.Finish()

		pn := gen.PulseNumber()
		pulseFetcher.FetchMock.Inspect(func(ctx context.Context, p1 insolar.PulseNumber) {
			assert.Equal(t, pn-10, p1)
		}).Return(&observer.Pulse{
			Number: pn,
		}, nil)

		recordFetcher.FetchMock.Inspect(func(ctx context.Context, pulse insolar.PulseNumber) {
			assert.Equal(t, pn, pulse)
		}).Return(nil, gen.PulseNumber(), errors.New("test"))

		s := state{
			last: pn - 10,
		}
		fetcher := makeFetcher(obs, pulseFetcher, recordFetcher)
		raw := fetcher(ctx, &s)

		assert.Nil(t, raw)
	})

	t.Run("Failed fetch curr heavy pulse", func(t *testing.T) {
		resetComponents()
		defer mc.Finish()

		pn := gen.PulseNumber()
		pulseFetcher.FetchMock.Inspect(func(ctx context.Context, p1 insolar.PulseNumber) {
			assert.Equal(t, pn-10, p1)
		}).Return(&observer.Pulse{
			Number: pn,
		}, nil)

		rec := map[uint32]*exporter.Record{
			0: {Record: record.Material{
				Polymorph: 0,
				Virtual:   record.Virtual{},
				ID:        insolar.ID{},
				ObjectID:  insolar.ID{},
				JetID:     insolar.JetID{},
				Signature: nil,
			}},
		}
		recordFetcher.FetchMock.Inspect(func(ctx context.Context, pulse insolar.PulseNumber) {
			assert.Equal(t, pn, pulse)
		}).Return(rec, gen.PulseNumber(), nil)

		pulseFetcher.FetchCurrentMock.Return(insolar.GenesisPulse.PulseNumber, errors.New("test"))

		s := state{
			last: pn - 10,
		}
		fetcher := makeFetcher(obs, pulseFetcher, recordFetcher)
		raw := fetcher(ctx, &s)
		assert.Equal(t, pn, raw.pulse.Number)
		assert.Equal(t, rec, raw.batch)
	})
}
