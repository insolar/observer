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

package api

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/observer/internal/models"
)

func TestNextRelease(t *testing.T) {
	for _, tc := range []struct {
		name        string
		deposit     models.Deposit
		currentTime int64
		amount      *big.Int
		expectation *SchemaNextRelease
	}{
		{
			name: "HoldReleaseDate 0",
			deposit: models.Deposit{
				HoldReleaseDate: 0,
				Amount:          "4000",
				Vesting:         31636000,
				VestingStep:     86400,
			},
			currentTime: 1573910366,
			amount:      big.NewInt(4000),
		},
		{
			name: "vesting 0",
			deposit: models.Deposit{
				HoldReleaseDate: 1573823965,
				Amount:          "4000",
				Vesting:         0,
				VestingStep:     86400,
			},
			currentTime: 1573910366,
			amount:      big.NewInt(4000),
		},
		{
			name: "vested",
			deposit: models.Deposit{
				HoldReleaseDate: 1573823965,
				Amount:          "4000",
				Vesting:         86400,
				VestingStep:     86400,
			},
			currentTime: 1573910365,
			amount:      big.NewInt(4000),
		},
		{
			name: "step > vested",
			deposit: models.Deposit{
				HoldReleaseDate: 1573823965,
				Amount:          "4000",
				Vesting:         86400,
				VestingStep:     86401,
			},
			currentTime: 1573910365,
			amount:      big.NewInt(4000),
		},
		{
			name: "before first step",
			deposit: models.Deposit{
				HoldReleaseDate: 1573823965,
				Amount:          "4000",
				Vesting:         31636000,
				VestingStep:     86400,
			},
			currentTime: 1573823964,
			amount:      big.NewInt(4000),
			expectation: &SchemaNextRelease{
				Amount:    "10",
				Timestamp: 1573910365,
			},
		},
		{
			name: "first step",
			deposit: models.Deposit{
				HoldReleaseDate: 1573823965,
				Amount:          "4000",
				Vesting:         31636000,
				VestingStep:     86400,
			},
			currentTime: 1573823967,
			amount:      big.NewInt(4000),
			expectation: &SchemaNextRelease{
				Amount:    "10",
				Timestamp: 1573910365,
			},
		},
		{
			name: "second step",
			deposit: models.Deposit{
				HoldReleaseDate: 1573823965,
				Amount:          "4000",
				Vesting:         31636000,
				VestingStep:     86400,
			},
			currentTime: 1573910366,
			amount:      big.NewInt(4000),
			expectation: &SchemaNextRelease{
				Amount:    "10",
				Timestamp: 1573996765,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := NextRelease(tc.currentTime, tc.amount, tc.deposit)
			assert.Equal(t, tc.expectation, res)
		})
	}
}
