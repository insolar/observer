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
	"math/big"
	"strconv"

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

func MemberToAPIMember(member models.Member, deposits []models.Deposit, currentTime int64, withMemberRef bool) ResponsesMemberYaml {
	var resDeposits []SchemaDeposit

	for _, d := range deposits {
		amount, _ := strconv.ParseInt(d.Amount, 10, 64)
		releaseAmount := d.ReleaseAmount(currentTime)
		balance, _ := strconv.ParseInt(d.Balance, 10, 64)
		amountOnHold := amount - releaseAmount
		resDeposit := SchemaDeposit{
			Index:           float32(d.DepositNumber),
			AmountOnHold:    strconv.FormatInt(amountOnHold, 10),
			AvailableAmount: strconv.FormatInt(balance-amountOnHold, 10),
			EthTxHash:       d.EtheriumHash,
			HoldReleaseDate: d.HoldReleaseDate,
			NextRelease:     NextRelease(currentTime, amount, d),
			ReleasedAmount:  strconv.FormatInt(releaseAmount, 10),
			ReleaseEndDate:  d.Vesting + d.HoldReleaseDate,
			Status:          d.Status(currentTime),
			Timestamp:       d.TransferDate,
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
		Balance: member.Balance,
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

	return res
}

func NextRelease(currentTime int64, amount int64, deposit models.Deposit) *SchemaNextRelease {
	var timestamp int64
	if deposit.HoldReleaseDate == 0 {
		return nil
	}
	if deposit.Vesting == 0 {
		return nil
	}
	if currentTime <= deposit.HoldReleaseDate {
		timestamp = deposit.HoldReleaseDate + deposit.VestingStep
	} else {
		if currentTime >= (deposit.HoldReleaseDate + deposit.Vesting) {
			return nil
		}
		timestamp = deposit.HoldReleaseDate + deposit.VestingStep*(((currentTime-deposit.HoldReleaseDate)/deposit.VestingStep)+1)
	}
	res := SchemaNextRelease{Timestamp: timestamp, Amount: nextReleaseAmount(amount, &deposit)}
	return &res
}

func nextReleaseAmount(amount int64, deposit *models.Deposit) string {
	stepValue := float64(deposit.VestingStep) / float64(deposit.Vesting)
	amountFloat := big.NewFloat(float64(amount))
	releaseAmount := new(big.Float).Mul(big.NewFloat(stepValue), amountFloat)
	return releaseAmount.String()
}
