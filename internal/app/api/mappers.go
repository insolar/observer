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
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/observer/internal/app/api/observerapi"
	"github.com/insolar/observer/internal/models"
)

func NullableString(s string) *string {
	return &s
}
func NullableInterface(i interface{}) *interface{} {
	return &i
}

func TxToAPITx(txID insolar.ID, tx models.Transaction) interface{} {
	internalTx := observerapi.SchemasTransactionAbstract{
		Amount:      tx.Amount,
		Fee:         NullableString(tx.Fee),
		Index:       "0",
		PulseNumber: tx.PulseNumber(),
		Status:      string(tx.Status()),
		Timestamp:   0,
		TxID:        txID.String(),
		Type:        string(tx.Type),
	}

	switch tx.Type {
	case models.TTypeMigration:
		res := observerapi.SchemaMigration{
			SchemasTransactionAbstract: internalTx,
			Type:                       NullableString(string(tx.Type)),
		}
		if len(tx.MemberFromReference) > 0 {
			ref := insolar.NewIDFromBytes(tx.MemberFromReference)
			res.FromMemberReference = NullableString(ref.String())
		}
		if len(tx.DepositToReference) > 0 {
			ref := insolar.NewIDFromBytes(tx.DepositToReference)
			res.ToDepositReference = NullableString(ref.String())
		}
		if len(tx.MemberToReference) > 0 {
			ref := insolar.NewIDFromBytes(tx.MemberToReference)
			res.ToMemberReference = NullableString(ref.String())
		}

		return res
	case models.TTypeTransfer:
		res := observerapi.SchemaTransfer{
			SchemasTransactionAbstract: internalTx,
			FromMemberReference:        NullableString(string(tx.MemberFromReference)),
			ToMemberReference:          NullableString(string(tx.MemberToReference)),
			Type:                       NullableString(string(tx.Type)),
		}
		if len(tx.MemberFromReference) > 0 {
			ref := insolar.NewIDFromBytes(tx.MemberFromReference)
			res.FromMemberReference = NullableString(ref.String())
		}
		if len(tx.MemberToReference) > 0 {
			ref := insolar.NewIDFromBytes(tx.MemberToReference)
			res.ToMemberReference = NullableString(ref.String())
		}
		return res
	case models.TTypeRelease:
		res := observerapi.SchemaRelease{
			SchemasTransactionAbstract: internalTx,
			FromDepositReference:       NullableString(string(tx.DepositFromReference)),
			ToMemberReference:          NullableString(string(tx.MemberToReference)),
			Type:                       NullableString(string(tx.Type)),
		}

		if len(tx.DepositFromReference) > 0 {
			ref := insolar.NewIDFromBytes(tx.DepositFromReference)
			res.FromDepositReference = NullableString(ref.String())
		}
		if len(tx.MemberToReference) > 0 {
			ref := insolar.NewIDFromBytes(tx.MemberToReference)
			res.ToMemberReference = NullableString(ref.String())
		}
		return res
	default:
		return internalTx
	}
}
