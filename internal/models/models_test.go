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

package models

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeposit_ReleaseAmount(t *testing.T) {
	for _, tc := range []struct {
		name             string
		deposit          Deposit
		currentTime      int64
		expectedHold     *big.Int
		expectedReleased *big.Int
	}{
		{
			name: "HoldReleaseDate = 0",
			deposit: Deposit{
				Balance:         "400000000",
				Amount:          "400000000",
				Vesting:         86400 * 1826,
				VestingStep:     86400,
				HoldReleaseDate: 0,
			},
			currentTime:      1574159397,
			expectedHold:     big.NewInt(0),
			expectedReleased: big.NewInt(400000000),
		},
		{
			name: "currentTime <= d.HoldReleaseDate",
			deposit: Deposit{
				Balance:         "400000000",
				Amount:          "400000000",
				Vesting:         86400 * 1826,
				VestingStep:     86400,
				HoldReleaseDate: 1574072996,
			},
			currentTime:      1574072995,
			expectedHold:     big.NewInt(400000000),
			expectedReleased: big.NewInt(0),
		},
		{
			name: "currentTime >= (d.Vesting + d.HoldReleaseDate)",
			deposit: Deposit{
				Balance:         "400000000",
				Amount:          "400000000",
				Vesting:         86400 * 1826,
				VestingStep:     86400,
				HoldReleaseDate: 1574072996,
			},
			currentTime:      1574072996 + 86400*2000,
			expectedHold:     big.NewInt(0),
			expectedReleased: big.NewInt(400000000),
		},
		{
			name: "step 0",
			deposit: Deposit{
				Balance:         "400000000",
				Amount:          "400000000",
				Vesting:         86400 * 1826,
				VestingStep:     86400,
				HoldReleaseDate: 1574072996,
			},
			currentTime:      1574072997,
			expectedHold:     big.NewInt(399999098),
			expectedReleased: big.NewInt(902),
		},
		{
			name: "step 0 with 0 balance",
			deposit: Deposit{
				Balance:         "0",
				Amount:          "400000000",
				Vesting:         86400 * 1826,
				VestingStep:     86400,
				HoldReleaseDate: 1574072996,
			},
			currentTime:      1574072997,
			expectedHold:     big.NewInt(0),
			expectedReleased: big.NewInt(0),
		},
		{
			name: "step 1",
			deposit: Deposit{
				Balance:         "400000000",
				Amount:          "400000000",
				Vesting:         86400 * 1826,
				VestingStep:     86400,
				HoldReleaseDate: 1574072996,
			},
			currentTime:      1574159397,
			expectedHold:     big.NewInt(399999098),
			expectedReleased: big.NewInt(902),
		},
		{
			name: "complex step 1",
			deposit: Deposit{
				Balance:         "499995000",
				Amount:          "500000000",
				Vesting:         18260,
				VestingStep:     10,
				HoldReleaseDate: 1606435200,
			},
			currentTime:      1606435311,
			expectedHold:     big.NewInt(499991926),
			expectedReleased: big.NewInt(8074),
		},
		{
			name: "complex step 2",
			deposit: Deposit{
				Balance:         "70000000",
				Amount:          "70000000",
				Vesting:         18260,
				VestingStep:     10,
				HoldReleaseDate: 1606435200,
			},
			currentTime:      1606435250,
			expectedHold:     big.NewInt(69999362),
			expectedReleased: big.NewInt(638),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			amount := new(big.Int)
			_, err := fmt.Sscan(tc.deposit.Amount, amount)
			require.NoError(t, err)
			balance := new(big.Int)
			_, err = fmt.Sscan(tc.deposit.Balance, balance)
			require.NoError(t, err)
			hold, release := tc.deposit.ReleaseAmount(balance, amount, tc.currentTime)
			assert.Equal(t, tc.expectedHold, hold)
			assert.Equal(t, tc.expectedReleased, release)
		})
	}
}
