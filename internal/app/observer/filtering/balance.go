// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package filtering

import (
	"github.com/insolar/insolar/insolar"

	"github.com/insolar/observer/internal/app/observer"
)

type BalanceFilter struct{}

func NewBalanceFilter() *BalanceFilter {
	return &BalanceFilter{}
}

func (*BalanceFilter) Filter(balances map[insolar.ID]*observer.Balance, members map[insolar.ID]*observer.Member) {
	// This code block collapses the balance sequence.
	for state, balance := range balances {
		bal, ok := balances[balance.PrevState]
		for ok {
			delete(balances, balance.PrevState)

			balance.PrevState = bal.PrevState
			bal, ok = balances[balance.PrevState]
		}
		balances[state] = balance
	}

	// We try to apply balance update in memory.
	for id, balance := range balances {
		m, ok := members[balance.PrevState]
		if !ok {
			continue
		}
		delete(members, balance.PrevState)
		m.Balance = balance.Balance
		m.AccountState = balance.AccountState
		members[balance.AccountState] = m
		delete(balances, id)
	}
}
