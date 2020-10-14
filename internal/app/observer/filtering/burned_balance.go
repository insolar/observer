// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

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
			bal, ok = burnedBalances[burnedBalance.PrevState]
		}
		burnedBalances[state] = burnedBalance
	}
}
