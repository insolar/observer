// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package filtering

import (
	"github.com/insolar/insolar/insolar"

	"github.com/insolar/observer/internal/app/observer"
)

type DepositUpdateFilter struct{}

func NewDepositUpdateFilter() *DepositUpdateFilter {
	return &DepositUpdateFilter{}
}

func (*DepositUpdateFilter) Filter(
	updates map[insolar.ID]observer.DepositUpdate,
	deposits map[insolar.ID]observer.Deposit) {
	// This code block collapses the deposit update sequence.
	for state, update := range updates {
		upd, ok := updates[update.PrevState]
		for ok {
			delete(updates, update.PrevState)

			update.PrevState = upd.PrevState
			upd, ok = updates[update.PrevState]
		}
		updates[state] = update
	}

	// We try to apply deposit update in memory.
	for id, update := range updates {
		d, ok := deposits[update.PrevState]
		if !ok {
			continue
		}
		delete(deposits, update.PrevState)
		d.Balance = update.Balance
		d.Amount = update.Amount
		d.HoldReleaseDate = update.HoldReleaseDate
		if update.HoldReleaseDate > update.Lockup {
			d.Timestamp = update.HoldReleaseDate - update.Lockup
		}
		d.DepositState = update.ID
		d.IsConfirmed = update.IsConfirmed
		deposits[update.ID] = d
		delete(updates, id)
	}
}
