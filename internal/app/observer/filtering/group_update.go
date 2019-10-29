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

type GroupUpdateFilter struct{}

func NewGroupUpdateFilter() *GroupUpdateFilter {
	return &GroupUpdateFilter{}
}

func (*GroupUpdateFilter) Filter(groupUpdates map[insolar.Reference]*observer.GroupUpdate, groups map[insolar.Reference]*observer.Group) {
	// This code block collapses the group update sequence.
	for _, update := range groupUpdates {
		upd, ok := groupUpdates[update.GroupReference]
		for ok {
			delete(groupUpdates, update.GroupReference)

			update.PrevState = upd.PrevState
			upd, ok = groupUpdates[update.GroupReference]
		}
	}

	// We try to apply deposit update in memory.
	for id, update := range groupUpdates {
		d, ok := groups[update.GroupReference]
		if !ok {
			continue
		}
		delete(groups, update.GroupReference)
		d.ProductType = update.ProductType
		d.Goal = update.Goal
		d.Image = update.Image
		d.State = update.GroupState
		d.Treasurer = update.Treasurer
		d.Title = update.Title
		d.ProductType = update.ProductType
		d.Timestamp = update.Timestamp
		d.Goal = update.Goal
		d.Membership = update.Membership
		d.ChairMan = update.ChairMan
		groups[update.GroupReference] = d
		delete(groupUpdates, id)
	}

}
