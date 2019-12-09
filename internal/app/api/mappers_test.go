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

	type variant struct {
		currentTime int64
		expectation *SchemaNextRelease
	}

	for _, tc := range []struct {
		name        string
		deposit     models.Deposit
		amount      *big.Int
		variants    []variant
	}{
		{
			name: "HoldReleaseDate 0",
			deposit: models.Deposit{
				HoldReleaseDate: 0,
				Amount:          "4000",
				Vesting:         0,
				VestingStep:     0,
			},
			amount: big.NewInt(4000),
			variants: []variant{
				{
					currentTime: 1573910366,
				},
			},
		},
		{
			name: "vesting 0",
			deposit: models.Deposit{
				HoldReleaseDate: 1573823965,
				Amount:          "4000",
				Vesting:         0,
				VestingStep:     0,
			},
			amount:      big.NewInt(4000),
			variants: []variant{
				{
					currentTime: 1573910366,
				},
			},
		},
		{
			name: "close to real numbers",
			deposit: models.Deposit{
				HoldReleaseDate: 1573823965,
				Amount:          "4000",
				Vesting:         31636000,
				VestingStep:     86400,
			},
			amount:      big.NewInt(4000),
			variants: []variant{
				{
					currentTime: 1573823964,
					expectation: &SchemaNextRelease{
						Amount:    "10",
						Timestamp: 1573910365,
					},
				},
				{
					currentTime: 1573823967,
					expectation: &SchemaNextRelease{
						Amount:    "10",
						Timestamp: 1573910365,
					},
				},
				{
					currentTime: 1573910366,
					expectation: &SchemaNextRelease{
						Amount:    "11",
						Timestamp: 1573996765,
					},
				},
				{
					currentTime: 1605459965,
				},
			},
		},
		{
			name: "jumping sum",
			deposit: models.Deposit{
				HoldReleaseDate: 100,
				Amount:          "6",
				Vesting:         40,
				VestingStep:     10,
			},
			amount:      big.NewInt(6),
			variants: []variant{
				{
					currentTime: 10,
					expectation: &SchemaNextRelease{
						Amount:    "1",
						Timestamp: 110,
					},
				},
				{
					currentTime: 20,
					expectation: &SchemaNextRelease{
						Amount:    "1",
						Timestamp: 110,
					},
				},
				{
					currentTime: 90,
					expectation: &SchemaNextRelease{
						Amount:    "1",
						Timestamp: 110,
					},
				},
				{
					currentTime: 100,
					expectation: &SchemaNextRelease{
						Amount:    "1",
						Timestamp: 110,
					},
				},
				{
					currentTime: 109,
					expectation: &SchemaNextRelease{
						Amount:    "1",
						Timestamp: 110,
					},
				},
				{
					currentTime: 110,
					expectation: &SchemaNextRelease{
						Amount:    "2",
						Timestamp: 120,
					},
				},
				{
					currentTime: 119,
					expectation: &SchemaNextRelease{
						Amount:    "2",
						Timestamp: 120,
					},
				},
				{
					currentTime: 120,
					expectation: &SchemaNextRelease{
						Amount:    "1",
						Timestamp: 130,
					},
				},
				{
					currentTime: 131,
					expectation: &SchemaNextRelease{
						Amount:    "2",
						Timestamp: 140,
					},
				},
				{
					currentTime: 140,
				},
			},
		},
		{
			name: "4k in 7steps",
			deposit: models.Deposit{
				HoldReleaseDate: 100,
				Amount:          "4000",
				Vesting:         70,
				VestingStep:     10,
			},
			amount:      big.NewInt(4000),
			variants: []variant{
				{
					currentTime: 90,
					expectation: &SchemaNextRelease{
						Amount:    "571",
						Timestamp: 110,
					},
				},
				{
					currentTime: 100,
					expectation: &SchemaNextRelease{
						Amount:    "571",
						Timestamp: 110,
					},
				},
				{
					currentTime: 110,
					expectation: &SchemaNextRelease{
						Amount:    "571",
						Timestamp: 120,
					},
				},
				{
					currentTime: 120,
					expectation: &SchemaNextRelease{
						Amount:    "572",
						Timestamp: 130,
					},
				},
				{
					currentTime: 130,
					expectation: &SchemaNextRelease{
						Amount:    "571",
						Timestamp: 140,
					},
				},
				{
					currentTime: 140,
					expectation: &SchemaNextRelease{
						Amount:    "572",
						Timestamp: 150,
					},
				},
				{
					currentTime: 150,
					expectation: &SchemaNextRelease{
						Amount:    "571",
						Timestamp: 160,
					},
				},
				{
					currentTime: 160,
					expectation: &SchemaNextRelease{
						Amount:    "572",
						Timestamp: 170,
					},
				},
				{
					currentTime: 170,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, e := range tc.variants {
				res := NextRelease(e.currentTime, tc.amount, tc.deposit)
				assert.Equal(t, e.expectation, res)
			}
		})
	}
}
