// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

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
				ID:              secondUpdate,
				Amount:          "120",
				Balance:         "123",
				IsConfirmed:     true,
				PrevState:       firstUpdate,
				Lockup:          10,
				HoldReleaseDate: now,
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
				Timestamp:       now - 10,
				HoldReleaseDate: now,
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
