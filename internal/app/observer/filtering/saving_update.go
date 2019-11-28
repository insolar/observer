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

type SavingUpdateFilter struct{}

func NewSavingUpdateFilter() *SavingUpdateFilter {
	return &SavingUpdateFilter{}
}

func (*SavingUpdateFilter) Filter(savingUpdates map[insolar.ID]*observer.SavingUpdate, savings map[insolar.ID]*observer.NormalSaving) {
	// This code block collapses the saving update sequence.
	for _, update := range savingUpdates {
		upd, ok := savingUpdates[update.PrevState]
		for ok {
			delete(savingUpdates, update.PrevState)

			update.PrevState = upd.PrevState
			upd, ok = savingUpdates[update.PrevState]
		}
	}

	// We try to apply saving update in memory.
	for id, update := range savingUpdates {
		s, ok := savings[update.PrevState]
		if !ok {
			continue
		}
		delete(savings, update.PrevState)
		s.StartRoundDate = update.StartRoundDate
		s.NextPaymentDate = update.NextPaymentDate
		s.NSContribute = update.NSContribute
		s.State = update.SavingState
		savings[update.SavingState] = s
		delete(savingUpdates, id)
	}

}
