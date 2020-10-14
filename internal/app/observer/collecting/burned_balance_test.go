// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package collecting

import (
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/pulse"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
)

func makeBurnAccountActivate(pn insolar.PulseNumber, balance string) *observer.Record {
	acc := &BurnAccount{
		Balance: balance,
	}
	memory, err := insolar.Serialize(acc)
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &record.Activate{
					Request:     *insolar.NewReference(gen.IDWithPulse(pn)),
					Memory:      memory,
					Image:       BurnAccountPrototypeReference,
					IsPrototype: false,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeBurnAccountAmend(pn insolar.PulseNumber, balance string, prev insolar.ID) *observer.Record {
	acc := &BurnAccount{
		Balance: balance,
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
					Image:     BurnAccountPrototypeReference,
					PrevState: prev,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeBurnBalanceActivate() (*observer.BurnedBalance, *observer.Record) {
	pn := insolar.PulseNumber(pulse.OfNow())
	balance := "12345678"
	rec := makeBurnAccountActivate(pn, balance)
	activate := &observer.BurnedBalance{
		AccountState: rec.ID,
		IsActivate:   true,
		Balance:      balance,
	}
	return activate, rec
}

func makeBurnBalanceUpdate() (*observer.BurnedBalance, *observer.Record) {
	pn := insolar.PulseNumber(pulse.OfNow())
	balance := "12345678"
	prev := gen.IDWithPulse(pn)
	rec := makeBurnAccountAmend(pn, balance, prev)
	update := &observer.BurnedBalance{
		PrevState:    prev,
		AccountState: rec.ID,
		IsActivate:   false,
		Balance:      balance,
	}
	return update, rec
}

func TestBurnedBalanceCollector_Collect(t *testing.T) {
	log := inslogger.FromContext(inslogger.TestContext(t))
	collector := NewBurnedBalanceCollector(log)

	t.Run("nil", func(t *testing.T) {
		require.Nil(t, collector.Collect(nil))
	})

	t.Run("non_amend", func(t *testing.T) {
		empty := insolar.ID{}
		require.Nil(t, collector.Collect(makeResultWith(empty, nil)))
	})

	t.Run("non_burned_account_amend", func(t *testing.T) {
		pn := pulse.OfNow()
		empty := insolar.ID{}
		require.Nil(t, collector.Collect(makeDepositAmend(pn, pn, "", "", empty)))
	})

	t.Run("ordinary_activate", func(t *testing.T) {
		activate, rec := makeBurnBalanceActivate()
		actual := collector.Collect(rec)

		require.Equal(t, activate, actual)
	})

	t.Run("ordinary_update", func(t *testing.T) {
		update, rec := makeBurnBalanceUpdate()
		actual := collector.Collect(rec)

		require.Equal(t, update, actual)
	})
}
