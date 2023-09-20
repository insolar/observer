package filtering

import (
	"github.com/insolar/insolar/insolar"

	"github.com/insolar/observer/internal/app/observer"
)

type BurnedBalanceFilter struct{}

func NewBurnedBalanceFilter() *BurnedBalanceFilter {
	return &BurnedBalanceFilter{}
}

func (*BurnedBalanceFilter) Filter(burnedBalances map[insolar.ID]*observer.BurnedBalance) {
	// This code block collapses the burn balance sequence.
	for state, burnedBalance := range burnedBalances {
		bal, ok := burnedBalances[burnedBalance.PrevState]
		for ok {
			delete(burnedBalances, burnedBalance.PrevState)

			burnedBalance.PrevState = bal.PrevState
			burnedBalance.IsActivate = bal.IsActivate
			bal, ok = burnedBalances[burnedBalance.PrevState]
		}
		burnedBalances[state] = burnedBalance
	}
}
