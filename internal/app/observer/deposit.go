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

package observer

import (
	"github.com/insolar/insolar/insolar"
)

type Deposit struct {
	EthHash         string
	Ref             insolar.Reference
	Member          insolar.Reference
	Timestamp       int64
	HoldReleaseDate int64
	Amount          string
	Balance         string
	DepositState    insolar.ID
	Vesting         int64
	VestingStep     int64
	DepositNumber   int64
}

type DepositCollector interface {
	Collect(*Record) *Deposit
}

type DepositUpdate struct {
	ID              insolar.ID
	HoldReleaseDate int64
	Amount          string
	Balance         string
	// Prev state record ID
	PrevState   insolar.ID
	TxHash      string // for debug purposes
	IsConfirmed bool
}

type DepositUpdateCollector interface {
	Collect(*Record) *DepositUpdate
}
