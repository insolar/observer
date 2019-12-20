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
	"context"
	"testing"

	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
)

func TestDepositMemberCollector_Collect(t *testing.T) {
	ctx := context.Background()
	log := inslogger.FromContext(inslogger.TestContext(t))
	collector := NewDepositMemberCollector(log)

	t.Run("confirm incoming request", func(t *testing.T) {
		pn := insolar.GenesisPulse.PulseNumber
		depositRef := gen.Reference()
		memberRef := gen.Reference()

		rec := makeConfirmIncRequest(pn, depositRef, memberRef)
		actual := collector.Collect(ctx, rec)

		expected := &observer.DepositMemberUpdate{
			Ref: depositRef,
			Member: memberRef,
		}
		require.NotNil(t, actual)
		require.Equal(t, expected, actual)
	})

	t.Run("deposit.new incoming request", func(t *testing.T) {
		pn := insolar.GenesisPulse.PulseNumber

		rec := makeNewDepositIncRequest(pn)
		actual := collector.Collect(ctx, rec)

		require.Nil(t, actual)
	})
}

func makeConfirmIncRequest(pn insolar.PulseNumber, depositRef, memberRef insolar.Reference) *observer.Record {
	raw, err := insolar.Serialize([]interface{}{nil, nil, nil, nil, &memberRef})
	if err != nil {
		panic("failed to serialize raw")
	}

	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method: "Confirm",
					CallType: record.CTMethod,
					Prototype: proxyDeposit.PrototypeReference,
					Arguments: raw,
					Object: &depositRef,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}
