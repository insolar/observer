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
	"github.com/insolar/observer/internal/models"
)

func NullableString(s string) *string {
	return &s
}
func NullableInterface(i interface{}) *interface{} {
	return &i
}

func TxToAPITx(tx models.Transaction, indexType models.TxIndexType) interface{} {
	internalTx := SchemasTransactionAbstract{
		Amount:      tx.Amount,
		Fee:         NullableString(tx.Fee),
		Index:       tx.Index(indexType),
		PulseNumber: tx.PulseNumber(),
		Status:      string(tx.Status()),
		Timestamp:   tx.Timestamp(),
		TxID:        insolar.NewReferenceFromBytes(tx.TransactionID).String(),
		Type:        string(tx.Type),
	}

	switch tx.Type {
	case models.TTypeMigration:
		res := SchemaMigration{
			SchemasTransactionAbstract: internalTx,
			Type:                       string(tx.Type),
		}
		if len(tx.MemberFromReference) > 0 {
			ref := insolar.NewReferenceFromBytes(tx.MemberFromReference)
			res.FromMemberReference = ref.String()
		}
		if len(tx.DepositToReference) > 0 {
			ref := insolar.NewReferenceFromBytes(tx.DepositToReference)
			res.ToDepositReference = ref.String()
		}
		if len(tx.MemberToReference) > 0 {
			ref := insolar.NewReferenceFromBytes(tx.MemberToReference)
			res.ToMemberReference = ref.String()
		}

		return res
	case models.TTypeTransfer:
		res := SchemaTransfer{
			SchemasTransactionAbstract: internalTx,
			Type:                       string(tx.Type),
		}
		if len(tx.MemberFromReference) > 0 {
			ref := insolar.NewReferenceFromBytes(tx.MemberFromReference)
			res.FromMemberReference = ref.String()
		}
		if len(tx.MemberToReference) > 0 {
			ref := insolar.NewReferenceFromBytes(tx.MemberToReference)
			res.ToMemberReference = ref.String()
		}

		return res
	case models.TTypeRelease:
		res := SchemaRelease{
			SchemasTransactionAbstract: internalTx,
			Type:                       string(tx.Type),
		}
		if len(tx.DepositFromReference) > 0 {
			ref := insolar.NewReferenceFromBytes(tx.DepositFromReference)
			res.FromDepositReference = ref.String()
		}
		if len(tx.MemberToReference) > 0 {
			ref := insolar.NewReferenceFromBytes(tx.MemberToReference)
			res.ToMemberReference = ref.String()
		}

		return res
	default:
		return internalTx
	}
}
