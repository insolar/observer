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
		name     string
		deposit  models.Deposit
		amount   *big.Int
		variants []variant
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
			amount: big.NewInt(4000),
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
				Vesting:         86400 * 1826,
				VestingStep:     86400,
			},
			amount: big.NewInt(4000),
			variants: []variant{
				{
					currentTime: 1573823965 + 86400*1030, // 1 662 815 965
					expectation: &SchemaNextRelease{
						Amount:    "1",
						Timestamp: 1573823965 + 86400*1031, // 1 662 902 365
					},
				},
				{
					currentTime: 1573823965 + 86400*1583, // 1 710 595 165
					expectation: &SchemaNextRelease{
						Amount:    "6",
						Timestamp: 1573823965 + 86400*1584, // 1 710 681 565
					},
				},
				{
					currentTime: 1573823965 + 86400*1798, // 1 729 171 165
					expectation: &SchemaNextRelease{
						Amount:    "15",
						Timestamp: 1573823965 + 86400*1799, // 1 729 257 565
					},
				},
				{
					currentTime: 1573823965 + 86400*2000,
				},
			},
		},
		{
			name: "no scaling, 1826 points",
			deposit: models.Deposit{
				HoldReleaseDate: 100,
				Amount:          "5000000000",
				Vesting:         18260,
				VestingStep:     10,
			},
			amount: big.NewInt(5000000000),
			variants: []variant{
				{
					currentTime: 10,
					expectation: &SchemaNextRelease{
						Amount:    "11279",
						Timestamp: 100,
					},
				},
				{
					currentTime: 20,
					expectation: &SchemaNextRelease{
						Amount:    "11279",
						Timestamp: 100,
					},
				},
				{
					currentTime: 90,
					expectation: &SchemaNextRelease{
						Amount:    "11279",
						Timestamp: 100,
					},
				},
				{
					currentTime: 100,
					expectation: &SchemaNextRelease{
						Amount:    "11326",
						Timestamp: 110,
					},
				},
				{
					currentTime: 109,
					expectation: &SchemaNextRelease{
						Amount:    "11326",
						Timestamp: 110,
					},
				},
				{
					currentTime: 110,
					expectation: &SchemaNextRelease{
						Amount:    "11372",
						Timestamp: 120,
					},
				},
				{
					currentTime: 119,
					expectation: &SchemaNextRelease{
						Amount:    "11372",
						Timestamp: 120,
					},
				},
				{
					currentTime: 14850,
					expectation: &SchemaNextRelease{
						Amount:    "4868415",
						Timestamp: 14860,
					},
				},
				{
					currentTime: 18350,
					expectation: &SchemaNextRelease{
						Amount:    "20523042",
						Timestamp: 18360,
					},
				},
				{
					currentTime: 18360,
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
			amount: big.NewInt(4000),
			variants: []variant{
				{
					currentTime: 90,
					expectation: &SchemaNextRelease{
						Amount:    "0",
						Timestamp: 100,
					},
				},
				{
					currentTime: 100,
					expectation: &SchemaNextRelease{
						Amount:    "4",
						Timestamp: 110,
					},
				},
				{
					currentTime: 110,
					expectation: &SchemaNextRelease{
						Amount:    "12",
						Timestamp: 120,
					},
				},
				{
					currentTime: 120,
					expectation: &SchemaNextRelease{
						Amount:    "36",
						Timestamp: 130,
					},
				},
				{
					currentTime: 130,
					expectation: &SchemaNextRelease{
						Amount:    "106",
						Timestamp: 140,
					},
				},
				{
					currentTime: 140,
					expectation: &SchemaNextRelease{
						Amount:    "309",
						Timestamp: 150,
					},
				},
				{
					currentTime: 150,
					expectation: &SchemaNextRelease{
						Amount:    "905",
						Timestamp: 160,
					},
				},
				{
					currentTime: 160,
					expectation: &SchemaNextRelease{
						Amount:    "2628",
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
