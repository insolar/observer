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
	for _, balance := range balances {
		bal, ok := balances[balance.PrevState]
		for ok {
			delete(balances, balance.PrevState)

			balance.PrevState = bal.PrevState
			bal, ok = balances[balance.PrevState]
		}
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
