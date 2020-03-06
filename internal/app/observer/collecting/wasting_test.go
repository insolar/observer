// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package collecting

import (
	"context"
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
)

func makeGetMigrationAddressCall(pn insolar.PulseNumber) *observer.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}

	virtRecord := record.Wrap(&record.IncomingRequest{
		Method:    GetFreeMigrationAddress,
		Arguments: args,
	})

	rec := &record.Material{
		ID:      gen.IDWithPulse(pn),
		Virtual: virtRecord,
	}
	return (*observer.Record)(rec)
}

func TestWastingCollector_Collect(t *testing.T) {
	log := inslogger.FromContext(inslogger.TestContext(t))
	t.Run("nil", func(t *testing.T) {
		fetcher := store.NewRecordFetcherMock(t)
		collector := NewWastingCollector(log, fetcher)
		ctx := context.Background()
		require.Nil(t, collector.Collect(ctx, nil))
	})

	t.Run("ordinary", func(t *testing.T) {
		fetcher := store.NewRecordFetcherMock(t)
		collector := NewWastingCollector(log, fetcher)

		pn := insolar.GenesisPulse.PulseNumber
		address := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"
		out := makeOutgoingRequest()
		call := makeGetMigrationAddressCall(pn)

		records := []*observer.Record{
			makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
			makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{address, nil}}),
		}

		fetcher.RequestMock.Set(func(_ context.Context, reqID insolar.ID) (m1 record.Material, err error) {
			switch reqID {
			case out.ID:
				return record.Material(*out), nil
			case call.ID:
				return record.Material(*call), nil
			default:
				panic("unexpected call")
			}
		})

		expected := []*observer.Wasting{
			{
				Addr: address,
			},
		}

		ctx := context.Background()
		var actual []*observer.Wasting
		for _, r := range records {
			wasting := collector.Collect(ctx, r)
			if wasting != nil {
				actual = append(actual, wasting)
			}
		}

		require.Len(t, actual, 1)
		require.Equal(t, expected, actual)
	})
}
