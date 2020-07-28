// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package collecting

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/mainnet/application/builtin/contract/deposit"
	proxyDeposit "github.com/insolar/mainnet/application/builtin/proxy/deposit"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
)

func makeDepositAmend(pn, unholdPulse insolar.PulseNumber, balance, amount string, prev insolar.ID) *observer.Record {
	acc := &deposit.Deposit{
		Balance:            balance,
		Amount:             amount,
		PulseDepositUnHold: unholdPulse,
	}
	memory, err := insolar.Serialize(acc)
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_Amend{
				Amend: &record.Amend{
					Request:   *insolar.NewReference(gen.IDWithPulse(pn)),
					Memory:    memory,
					Image:     *proxyDeposit.PrototypeReference,
					PrevState: prev,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeDepositUpdate() (*observer.DepositUpdate, *observer.Record) {
	pn := insolar.GenesisPulse.PulseNumber
	unholdPulse := pn + 3
	amount := "4"
	balance := "3"
	prev := gen.IDWithPulse(pn)
	rec := makeDepositAmend(pn, unholdPulse, balance, amount, prev)
	timestamp, err := unholdPulse.AsApproximateTime()
	if err != nil {
		panic("invalid pulse")
	}
	upd := &observer.DepositUpdate{
		ID:              rec.ID,
		HoldReleaseDate: timestamp.Unix(),
		Timestamp:       timestamp.Unix(),
		Amount:          amount,
		Balance:         balance,
		PrevState:       prev,
	}
	return upd, rec
}

func TestDepositUpdateCollector_Collect(t *testing.T) {

	t.Run("nil", func(t *testing.T) {
		log := inslogger.FromContext(inslogger.TestContext(t))
		mc := minimock.NewController(t)

		collector := NewDepositUpdateCollector(log)

		ctx := context.Background()
		require.Nil(t, collector.Collect(ctx, nil))

		mc.Finish()
	})

	t.Run("ordinary", func(t *testing.T) {
		log := inslogger.FromContext(inslogger.TestContext(t))
		mc := minimock.NewController(t)

		collector := NewDepositUpdateCollector(log)

		ctx := context.Background()
		upd, rec := makeDepositUpdate()

		actual := collector.Collect(ctx, rec)
		require.Equal(t, upd, actual)

		mc.Finish()
	})
}
