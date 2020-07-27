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
	proxyDeposit "github.com/insolar/mainnet/application/builtin/proxy/deposit"
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
			Ref:    depositRef,
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
					Method:    "Confirm",
					CallType:  record.CTMethod,
					Prototype: proxyDeposit.PrototypeReference,
					Arguments: raw,
					Object:    &depositRef,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}
