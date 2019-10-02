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

import "github.com/insolar/insolar/insolar"

// Transfer describes token moving between the insolar members.
type Transfer struct {
	TxID      insolar.ID
	From      insolar.ID
	To        insolar.ID
	Amount    string
	Fee       string
	Timestamp int64
	Pulse     insolar.PulseNumber
}

// DepositTransfer describes token moving from deposit account to the insolar member account.
type DepositTransfer struct {
	Transfer
	EthHash string
}

type TransferStorage interface {
	Insert(*Transfer) error
}

type TransferCollector interface {
	Collect(*Record) *DepositTransfer
}
