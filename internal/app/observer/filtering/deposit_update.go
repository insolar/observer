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

type DepositUpdateFilter struct{}

func NewDepositUpdateFilter() *DepositUpdateFilter {
	return &DepositUpdateFilter{}
}

func (*DepositUpdateFilter) Filter(updates map[insolar.ID]observer.DepositUpdate, deposits map[insolar.ID]observer.Deposit) {
	// This code block collapses the deposit update sequence.
	for _, update := range updates {
		upd, ok := updates[update.PrevState]
		for ok {
			delete(updates, update.PrevState)

			update.PrevState = upd.PrevState
			upd, ok = updates[update.PrevState]
		}
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
		d.DepositState = update.ID
		d.IsConfirmed = update.IsConfirmed
		deposits[update.ID] = d
		delete(updates, id)
	}
}
