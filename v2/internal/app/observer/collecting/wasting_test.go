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

package collecting

import (
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/v2/internal/app/observer"
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
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "GetFreeMigrationAddress",
					Arguments: args,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeWasting() ([]*observer.Wasting, []*observer.Record) {
	pn := insolar.GenesisPulse.PulseNumber
	address := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"
	out := makeOutgouingRequest()
	call := makeGetMigrationAddressCall(pn)
	records := []*observer.Record{
		out,
		makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		call,
		makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{address, nil}}),
	}

	wasting := &observer.Wasting{
		Addr: address,
	}
	return []*observer.Wasting{wasting}, records
}

func TestWastingCollector_Collect(t *testing.T) {
	collector := NewWastingCollector()

	t.Run("nil", func(t *testing.T) {
		require.Nil(t, collector.Collect(nil))
	})

	t.Run("ordinary", func(t *testing.T) {
		expected, records := makeWasting()
		var actual []*observer.Wasting
		for _, r := range records {
			wasting := collector.Collect(r)
			if wasting != nil {
				actual = append(actual, wasting)
			}
		}

		require.Len(t, actual, 1)
		require.Equal(t, expected, actual)
	})
}
