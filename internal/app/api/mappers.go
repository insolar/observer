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

	depositContract "github.com/insolar/insolar/application/builtin/contract/deposit"
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

		// it's a hack - special case for deposits from which we transfer money for migration
		// (work for migration_admin_member's deposit)
		if currentTime < d.HoldReleaseDate && amountOnHold.Cmp(balance) == 1 {
			amountOnHold = balance
		}

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
			NextRelease:     NextRelease(currentTime, amount, balance, d),
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

func NextRelease(currentTime int64, amount *big.Int, balance *big.Int, deposit models.Deposit) *SchemaNextRelease {
	if deposit.HoldReleaseDate == 0 {
		return nil
	}

	if deposit.Vesting == 0 {
		return nil
	}

	if balance.Cmp(big.NewInt(0)) <= 0 {
		return nil
	}

	vestingEnd := deposit.HoldReleaseDate + deposit.Vesting
	if currentTime >= vestingEnd {
		return nil
	}

	var timestamp int64
	if currentTime < deposit.HoldReleaseDate {
		timestamp = deposit.HoldReleaseDate
	} else {
		timestamp = deposit.HoldReleaseDate + deposit.VestingStep*(((currentTime-deposit.HoldReleaseDate)/deposit.VestingStep)+1)
		if timestamp >= vestingEnd {
			return checkAgainstBalance(vestingEnd, balance, lastReleaseAmount(amount, &deposit))
		}
	}

	return checkAgainstBalance(timestamp, balance, nextReleaseAmount(amount, &deposit, currentTime))
}

func checkAgainstBalance(ts int64, balance *big.Int, next *big.Int) *SchemaNextRelease {
	if balance.Cmp(next) == -1 {
		next = balance
	}
	return &SchemaNextRelease{Timestamp: ts, Amount: next.Text(10)}
}

func nextReleaseAmount(amount *big.Int, deposit *models.Deposit, currentTime int64) *big.Int {
	steps := deposit.Vesting / deposit.VestingStep
	sinceRelease := currentTime - deposit.HoldReleaseDate
	if sinceRelease < 0 {
		return depositContract.VestedByNow(amount, 0, uint64(steps))
	}

	step := sinceRelease / deposit.VestingStep
	releasedAmount := depositContract.VestedByNow(amount, uint64(step), uint64(steps))
	willReleaseAmount := depositContract.VestedByNow(amount, uint64(step+1), uint64(steps))

	return new(big.Int).Sub(willReleaseAmount, releasedAmount)
}

func lastReleaseAmount(amount *big.Int, deposit *models.Deposit) *big.Int {
	steps := deposit.Vesting / deposit.VestingStep
	releasedAmount := depositContract.VestedByNow(amount, uint64(steps-1), uint64(steps))
	return new(big.Int).Sub(amount, releasedAmount)
}

func (response *ResponsesMarketStatsYaml) addHistoryPoints(points []models.PriceHistory) {
	var parsedPoints []struct {
		Price     string `json:"price"`
		Timestamp int64  `json:"timestamp"`
	}
	for _, point := range points {
		parsedPoints = append(parsedPoints, struct {
			Price     string `json:"price"`
			Timestamp int64  `json:"timestamp"`
		}{
			fmt.Sprintf("%v", point.Price),
			point.Timestamp.Unix(),
		})
	}
	response.PriceHistory = &parsedPoints
}
