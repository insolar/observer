// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package collecting

import (
	"testing"

	"github.com/insolar/insolar/application/builtin/contract/account"
	proxyAccount "github.com/insolar/insolar/application/builtin/proxy/account"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/pulse"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
)

func makeAccountAmend(pn insolar.PulseNumber, balance string, prev insolar.ID) *observer.Record {
	acc := &account.Account{
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
					Image:     *proxyAccount.PrototypeReference,
					PrevState: prev,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeBalanceUpdate() (*observer.Balance, *observer.Record) {
	pn := insolar.PulseNumber(pulse.OfNow())
	balance := "43"
	prev := gen.IDWithPulse(pn)
	rec := makeAccountAmend(pn, balance, prev)
	update := &observer.Balance{
		PrevState:    prev,
		AccountState: rec.ID,
		Balance:      balance,
	}
	return update, rec
}

func TestBalanceCollector_Collect(t *testing.T) {
	log := inslogger.FromContext(inslogger.TestContext(t))
	collector := NewBalanceCollector(log)

	t.Run("nil", func(t *testing.T) {
		require.Nil(t, collector.Collect(nil))
	})

	t.Run("non_amend", func(t *testing.T) {
		empty := insolar.ID{}
		require.Nil(t, collector.Collect(makeResultWith(empty, nil)))
	})

	t.Run("non_account_amend", func(t *testing.T) {
		pn := pulse.OfNow()
		empty := insolar.ID{}
		require.Nil(t, collector.Collect(makeDepositAmend(pn, pn, "", "", empty)))
	})

	t.Run("ordinary", func(t *testing.T) {
		update, rec := makeBalanceUpdate()
		actual := collector.Collect(rec)

		require.Equal(t, update, actual)
	})
}
