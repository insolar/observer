package component

import (
	"github.com/insolar/observer/internal/app/observer/filtering"
	"github.com/insolar/observer/observability"
)

func makeFilter(obs *observability.Observability) func(*beauty) *beauty {
	log := obs.Log()
	metric := observability.MakeBeautyMetrics(obs, "filtered")
	return func(b *beauty) *beauty {
		if b == nil {
			return nil
		}

		filtering.NewBalanceFilter().Filter(b.balances, b.members)
		filtering.NewDepositUpdateFilter().Filter(b.depositUpdates, b.deposits)
		filtering.NewVestingFilter().Filter(b.vestings, b.addresses)
		filtering.NewBurnedBalanceFilter().Filter(b.burnedBalances)

		log.Info("items successfully filtered")

		metric.Transfers.Add(float64(len(b.txSagaResult)))
		metric.Members.Add(float64(len(b.members)))
		metric.Deposits.Add(float64(len(b.deposits)))
		metric.Addresses.Add(float64(len(b.addresses)))

		metric.Balances.Add(float64(len(b.balances)))
		metric.Updates.Add(float64(len(b.depositUpdates)))
		metric.Vestings.Add(float64(len(b.vestings)))
		return b
	}
}
