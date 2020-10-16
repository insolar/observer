// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package api

import (
	"fmt"
	"math/big"

	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/models"
)

func NullableString(s string) *string {
	return &s
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
	case models.TTypeMigration, models.TTypeAllocation:
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
			refStr := insolar.NewReferenceFromBytes(tx.MemberToReference).String()
			// ToMemberReference should remain ref, because it is nullable in spec, for now I manually edited generated.go
			res.ToMemberReference = &refStr
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

func MemberToAPIMember(member models.Member, deposits []models.Deposit, burnedBalance *models.BurnedBalance, withMemberRef bool) (ResponsesMemberYaml, error) {
	var resDeposits []SchemaDeposit

	for _, d := range deposits {
		amount := new(big.Int)
		if _, err := fmt.Sscan(d.Amount, amount); err != nil {
			return ResponsesMemberYaml{}, errors.Wrap(err, "failed to parse deposit amount")
		}
		balance := new(big.Int)
		if _, err := fmt.Sscan(d.Balance, balance); err != nil {
			return ResponsesMemberYaml{}, errors.Wrap(err, "failed to parse deposit balance")
		}

		resDeposit := SchemaDeposit{
			Index:           int(*d.DepositNumber),
			AmountOnHold:    "0",
			AvailableAmount: balance.Text(10),
			EthTxHash:       d.EtheriumHash,
			HoldReleaseDate: d.Timestamp,
			ReleasedAmount:  amount.Text(10),
			ReleaseEndDate:  d.Timestamp,
			Status:          "AVAILABLE",
			Timestamp:       d.Timestamp,
		}
		ref := insolar.NewReferenceFromBytes(d.Reference)
		if ref != nil {
			resDeposit.DepositReference = ref.String()
		}
		if withMemberRef {
			ref := insolar.NewReferenceFromBytes(member.Reference)
			if ref != nil {
				resDeposit.MemberReference = NullableString(ref.String())
			}
		}

		resDeposits = append(resDeposits, resDeposit)
	}

	res := ResponsesMemberYaml{
		Balance:   member.Balance,
		Reference: insolar.NewReferenceFromBytes(member.Reference).String(),
	}
	if len(resDeposits) > 0 {
		res.Deposits = &resDeposits
	}
	ref := insolar.NewReferenceFromBytes(member.WalletReference)
	if ref != nil {
		res.WalletReference = ref.String()
	}
	ref = insolar.NewReferenceFromBytes(member.AccountReference)
	if ref != nil {
		res.AccountReference = ref.String()
	}
	if member.MigrationAddress != "" {
		res.MigrationAddress = &member.MigrationAddress
	}
	if burnedBalance != nil {
		res.BurnedBalance = NullableString(burnedBalance.Balance)
	}

	return res, nil
}
