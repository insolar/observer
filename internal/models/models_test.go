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
				Balance:         "400",
				Amount:          "400",
				Vesting:         31636000,
				VestingStep:     86400,
				HoldReleaseDate: 0,
			},
			currentTime:      1574159397,
			expectedHold:     big.NewInt(0),
			expectedReleased: big.NewInt(400),
		},
		{
			name: "currentTime <= d.HoldReleaseDate",
			deposit: Deposit{
				Balance:         "400",
				Amount:          "400",
				Vesting:         31636000,
				VestingStep:     86400,
				HoldReleaseDate: 1574072996,
			},
			currentTime:      1574072995,
			expectedHold:     big.NewInt(400),
			expectedReleased: big.NewInt(0),
		},
		{
			name: "currentTime >= (d.Vesting + d.HoldReleaseDate)",
			deposit: Deposit{
				Balance:         "400",
				Amount:          "400",
				Vesting:         31636000,
				VestingStep:     86400,
				HoldReleaseDate: 1574072996,
			},
			currentTime:      1605708997,
			expectedHold:     big.NewInt(0),
			expectedReleased: big.NewInt(400),
		},
		{
			name: "step 0",
			deposit: Deposit{
				Balance:         "400",
				Amount:          "400",
				Vesting:         31636000,
				VestingStep:     86400,
				HoldReleaseDate: 1574072996,
			},
			currentTime:      1574072997,
			expectedHold:     big.NewInt(400),
			expectedReleased: big.NewInt(0),
		},
		{
			name: "step 0 with 0 balance",
			deposit: Deposit{
				Balance:         "0",
				Amount:          "400",
				Vesting:         31636000,
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
				Balance:         "400",
				Amount:          "400",
				Vesting:         31636000,
				VestingStep:     86400,
				HoldReleaseDate: 1574072996,
			},
			currentTime:      1574159397,
			expectedHold:     big.NewInt(399),
			expectedReleased: big.NewInt(1),
		},
		{
			name: "complex step",
			deposit: Deposit{
				Balance:         "4500",
				Amount:          "5000",
				Vesting:         1000,
				VestingStep:     10,
				HoldReleaseDate: 1606435200,
			},
			currentTime:      1606435311,
			expectedHold:     big.NewInt(4450),
			expectedReleased: big.NewInt(550),
		},
		{
			name: "complex step",
			deposit: Deposit{
				Balance:         "70000",
				Amount:          "70000",
				Vesting:         70,
				VestingStep:     10,
				HoldReleaseDate: 1606435200,
			},
			currentTime:      1606435250,
			expectedHold:     big.NewInt(20000),
			expectedReleased: big.NewInt(50000),
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
			require.Equal(t, tc.expectedHold, hold)
			require.Equal(t, tc.expectedReleased, release)
		})
	}
}
