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
	"fmt"
	"math/big"

	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"

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

func MemberToAPIMember(member models.Member, deposits []models.Deposit, currentTime int64, withMemberRef bool) (ResponsesMemberYaml, error) {
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
		amountOnHold, releaseAmount := d.ReleaseAmount(balance, amount, currentTime)
		available := big.NewInt(0).Sub(balance, amountOnHold)
		// if partially vested and partially transferred to wallet
		if available.Cmp(big.NewInt(0)) == -1 {
			available = big.NewInt(0)
		}
		resDeposit := SchemaDeposit{
			Index:           int(*d.DepositNumber),
			AmountOnHold:    amountOnHold.Text(10),
			AvailableAmount: available.Text(10),
			EthTxHash:       d.EtheriumHash,
			HoldReleaseDate: d.HoldReleaseDate,
			NextRelease:     NextRelease(currentTime, amount, d),
			ReleasedAmount:  releaseAmount.Text(10),
			ReleaseEndDate:  d.Vesting + d.HoldReleaseDate,
			Status:          d.Status(currentTime),
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

	return res, nil
}

func NextRelease(currentTime int64, amount *big.Int, deposit models.Deposit) *SchemaNextRelease {
	if deposit.HoldReleaseDate == 0 {
		return nil
	}

	if deposit.Vesting == 0 {
		return nil
	}

	vestingEnd := deposit.HoldReleaseDate + deposit.Vesting
	if currentTime >= vestingEnd {
		return nil
	}

	var timestamp int64
	if currentTime <= deposit.HoldReleaseDate {
		timestamp = deposit.HoldReleaseDate + deposit.VestingStep
	} else {
		timestamp = deposit.HoldReleaseDate + deposit.VestingStep*(((currentTime-deposit.HoldReleaseDate)/deposit.VestingStep)+1)
		if timestamp >= vestingEnd {
			return &SchemaNextRelease{Timestamp: vestingEnd, Amount: lastReleaseAmount(amount, &deposit)}
		}
	}
	return &SchemaNextRelease{Timestamp: timestamp, Amount: nextReleaseAmount(amount, &deposit, currentTime)}
}

func nextReleaseAmount(amount *big.Int, deposit *models.Deposit, currentTime int64) string {
	steps := deposit.Vesting / deposit.VestingStep
	sinceRelease := currentTime - deposit.HoldReleaseDate
	if sinceRelease < 0 {
		sinceRelease = 0
	}
	step := sinceRelease / deposit.VestingStep
	releasedAmount := new(big.Int).Quo(new(big.Int).Mul(amount, big.NewInt(step)), big.NewInt(steps))
	willReleaseAmount := new(big.Int).Quo(new(big.Int).Mul(amount, big.NewInt(step+1)), big.NewInt(steps))

	return new(big.Int).Sub(willReleaseAmount, releasedAmount).Text(10)
}

func lastReleaseAmount(amount *big.Int, deposit *models.Deposit) string {
	steps := deposit.Vesting / deposit.VestingStep
	releasedAmount := new(big.Int).Quo(new(big.Int).Mul(amount, big.NewInt(steps-1)), big.NewInt(steps))
	return new(big.Int).Sub(amount, releasedAmount).Text(10)
}
