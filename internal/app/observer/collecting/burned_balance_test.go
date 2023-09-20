package collecting

import (
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/pulse"
	"github.com/insolar/mainnet/application/builtin/contract/burnedaccount"
	proxyBurned "github.com/insolar/mainnet/application/builtin/proxy/burnedaccount"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
)

func makeBurnAccountActivate(pn insolar.PulseNumber, balance string) *observer.Record {
	acc := &burnedaccount.BurnedAccount{
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
					Image:       *proxyBurned.PrototypeReference,
					IsPrototype: false,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeBurnAccountAmend(pn insolar.PulseNumber, balance string, prev insolar.ID) *observer.Record {
	acc := &burnedaccount.BurnedAccount{
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
					Image:     *proxyBurned.PrototypeReference,
					PrevState: prev,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeBurnBalance(isActivate bool) (*observer.BurnedBalance, *observer.Record) {
	pn := pulse.OfNow()
	balance := "12345678"
	burnedBalance := &observer.BurnedBalance{
		IsActivate: isActivate,
		Balance:    balance,
	}
	if isActivate {
		rec := makeBurnAccountActivate(pn, balance)
		burnedBalance.AccountState = rec.ID
		return burnedBalance, rec
	}
	prev := gen.IDWithPulse(pn)
	rec := makeBurnAccountAmend(pn, balance, prev)
	burnedBalance.AccountState = rec.ID
	burnedBalance.PrevState = prev
	return burnedBalance, rec
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
		activate, rec := makeBurnBalance(true)
		actual := collector.Collect(rec)

		require.Equal(t, activate, actual)
	})

	t.Run("ordinary_update", func(t *testing.T) {
		update, rec := makeBurnBalance(false)
		actual := collector.Collect(rec)

		require.Equal(t, update, actual)
	})
}
