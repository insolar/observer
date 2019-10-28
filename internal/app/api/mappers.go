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
	"github.com/insolar/observer/internal/models"
)

func NullableString(s string) *string {
	return &s
}
func NullableInterface(i interface{}) *interface{} {
	return &i
}

func TxToApiTx(txID string, tx models.Transaction) interface{} {
	internalTx := Transaction{
		Amount:      tx.Amount,
		Fee:         tx.Fee,
		Index:       0,
		PulseNumber: float32(tx.PulseNumber),
		Status:      string(tx.Status()),
		Timestamp:   0,
		TxID:        txID,
		Type:        string(tx.Type()),
	}

	switch tx.Type() {
	case models.TTypeMigration:
		return Migration{
			Transaction:         internalTx,
			FromMemberReference: NullableString(string(tx.MemberFromReference)),
			ToDepositReference:  NullableString(string(tx.MigrationsToReference)),
			ToMemberReference:   NullableString(string(tx.MemberToReference)),
			Type:                NullableInterface(tx.Type()),
		}
	case models.TTypeTransfer:
		return Transfer{
			Transaction:         internalTx,
			FromMemberReference: NullableString(string(tx.MemberFromReference)),
			ToMemberReference:   NullableString(string(tx.MemberToReference)),
			Type:                NullableInterface(tx.Type()),
		}
	case models.TTypeRelease:
		return Release{
			Transaction:          internalTx,
			FromDepositReference: NullableString(string(tx.VestingFromReference)),
			ToMemberReference:    NullableString(string(tx.MemberToReference)),
			Type:                 NullableInterface(tx.Type()),
		}
	default:
		return internalTx
	}
}