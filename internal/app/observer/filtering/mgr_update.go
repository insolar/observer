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

type MGRUpdateFilter struct{}

func NewMGRUpdateFilter() *MGRUpdateFilter {
	return &MGRUpdateFilter{}
}

func (*MGRUpdateFilter) Filter(mgrUpdates map[insolar.ID]*observer.MGRUpdate, mgrs map[insolar.ID]*observer.MGR) {
	// This code block collapses the mgr update sequence.
	for _, update := range mgrUpdates {
		upd, ok := mgrUpdates[update.PrevState]
		for ok {
			delete(mgrUpdates, update.PrevState)

			update.PrevState = upd.PrevState
			upd, ok = mgrUpdates[update.PrevState]
		}
	}

	// We try to apply mgr update in memory.
	for id, update := range mgrUpdates {
		d, ok := mgrs[update.PrevState]
		if !ok {
			continue
		}
		delete(mgrs, update.PrevState)
		d.SwapProcess = update.SwapProcess
		d.Sequence = update.Sequence
		d.NextPaymentTime = update.NextPaymentTime
		d.FinishRoundDate = update.FinishRoundDate
		d.StartRoundDate = update.StartRoundDate
		d.AmountDue = update.AmountDue
		d.PaymentFrequency = update.PaymentFrequency
		mgrs[update.MGRState] = d
		delete(mgrUpdates, id)
	}

}
