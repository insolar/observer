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

package component

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

func TestFilter_DepositsUpdates(t *testing.T) {
	obs := observability.Make(context.Background())

	filter := makeFilter(obs)

	initState := gen.ID()
	firstUpdate := gen.ID()
	secondUpdate := gen.ID()

	otherUpdate := gen.ID()
	otherPrevState := gen.ID()

	depositRef := gen.Reference()
	memberRef := gen.Reference()
	now := time.Now().Unix()

	input := &beauty{
		pulse: &observer.Pulse{
			Number: insolar.GenesisPulse.PulseNumber,
			Nodes: []insolar.Node{
				{
					ID:   gen.Reference(),
					Role: insolar.StaticRoleHeavyMaterial,
				},
			},
		},
		deposits: map[insolar.ID]observer.Deposit{
			initState: {
				EthHash:         strings.ToLower("0x5ca5e6417f818ba1c74d"),
				Ref:             depositRef,
				Member:          memberRef,
				Timestamp:       now,
				HoldReleaseDate: 0,
				Amount:          "120",
				Balance:         "123",
				DepositState:    initState,
				Vesting:         10,
				VestingStep:     10,
			},
		},
		depositUpdates: map[insolar.ID]observer.DepositUpdate{
			firstUpdate: {
				ID:          firstUpdate,
				Amount:      "120",
				Balance:     "123",
				IsConfirmed: false,
				PrevState:   initState,
			},
			secondUpdate: {
				ID:          secondUpdate,
				Amount:      "120",
				Balance:     "123",
				IsConfirmed: true,
				PrevState:   firstUpdate,
			},
			otherUpdate: {
				ID:          otherUpdate,
				Amount:      "120",
				Balance:     "123",
				IsConfirmed: true,
				PrevState:   otherPrevState,
			},
		},
	}

	filtered := filter(input)

	assert.Equal(t, &beauty{
		pulse: input.pulse,
		depositUpdates: map[insolar.ID]observer.DepositUpdate{otherUpdate: {
			ID:          otherUpdate,
			Amount:      "120",
			Balance:     "123",
			IsConfirmed: true,
			PrevState:   otherPrevState,
		}},
		deposits: map[insolar.ID]observer.Deposit{
			secondUpdate: {
				EthHash:         strings.ToLower("0x5ca5e6417f818ba1c74d"),
				Ref:             depositRef,
				Member:          memberRef,
				Timestamp:       now,
				HoldReleaseDate: 0,
				Amount:          "120",
				Balance:         "123",
				DepositState:    secondUpdate,
				Vesting:         10,
				VestingStep:     10,
				IsConfirmed:     true,
			},
		},
	}, filtered)
}
