// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package api

import (
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/models"
)

func TestAllocationTransactions(t *testing.T) {
	memberFrom := gen.Reference()
	memberTo := gen.Reference()
	depositTo := gen.Reference()
	tx := models.Transaction{
		Amount:              "100",
		Fee:                 *NullableString("0"),
		StatusRegistered:    true,
		TransactionID:       gen.Reference().Bytes(),
		Type:                models.TTypeAllocation,
		MemberFromReference: memberFrom.Bytes(),
		MemberToReference:   memberTo.Bytes(),
		DepositToReference:  depositTo.Bytes(),
	}
	indexType := models.TxIndexTypeFinishPulseRecord
	txMigration := SchemaMigration{
		SchemasTransactionAbstract: SchemasTransactionAbstract{
			Amount:      tx.Amount,
			Fee:         NullableString(tx.Fee),
			Index:       tx.Index(indexType),
			PulseNumber: tx.PulseNumber(),
			Status:      string(tx.Status()),
			Timestamp:   tx.Timestamp(),
			TxID:        insolar.NewReferenceFromBytes(tx.TransactionID).String(),
			Type:        string(tx.Type),
		},
		Type:                string(tx.Type),
		FromMemberReference: memberFrom.String(),
		ToMemberReference:   memberTo.String(),
		ToDepositReference:  depositTo.String(),
	}
	res := TxToAPITx(tx, indexType)
	require.Equal(t, txMigration, res.(SchemaMigration))
}
