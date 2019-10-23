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

type TransferStatus int

const (
	Pending TransferStatus = iota + 1
	Success
	Failed
)

func (s *TransferStatus) String() string {
	switch *s {
	case Pending:
		return "PENDING"
	case Success:
		return "SUCCESS"
	case Failed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

type TransferKind int

const (
	Migration TransferKind = iota + 1
	Withdraw
	Standard
)

type TransferDirection int

const (
	APICall TransferDirection = iota + 1
	Saga
)

// Transfer describes token moving between the insolar members,
// migration from ethereum INS tokens to insolar XNS (to deposit)
// and withdrawal from deposit to account.
type Transfer struct {
	TxID          insolar.ID // api.Call ID or saga request ID
	From          *insolar.ID
	To            *insolar.ID
	Amount        string
	Fee           string
	EthHash       string
	Status        TransferStatus    // pending / success / failed
	Kind          TransferKind      // migration / withdraw / standard
	Direction     TransferDirection // api.Call / saga
	DetachRequest *insolar.ID       // final request before saga call
	ParentRequest *insolar.ID       // optional, notnil if it is saga
	Details       string            // json with extra info
}

type TransferStorage interface {
	Insert(*Transfer) error
}

type TransferCollector interface {
	Collect(*Record) *Transfer
}
