// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package api

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/observer/internal/models"
)

func TestNextRelease(t *testing.T) {

	type variant struct {
		currentTime int64
		expectation *SchemaNextRelease
	}

	for _, tc := range []struct {
		name     string
		deposit  models.Deposit
		amount   *big.Int
		balance  *big.Int
		variants []variant
	}{
		{
			name: "HoldReleaseDate 0",
			deposit: models.Deposit{
				HoldReleaseDate: 0,
				Amount:          "4000",
				Vesting:         0,
				VestingStep:     0,
			},
			amount: big.NewInt(4000),
			variants: []variant{
				{
					currentTime: 1573910366,
				},
			},
		},
		{
			name: "vesting 0",
			deposit: models.Deposit{
				HoldReleaseDate: 1573823965,
				Amount:          "4000",
				Vesting:         0,
				VestingStep:     0,
			},
			amount: big.NewInt(4000),
			variants: []variant{
				{
					currentTime: 1573910366,
				},
			},
		},
		{
			name: "close to real numbers",
			deposit: models.Deposit{
				HoldReleaseDate: 1573823965,
				Amount:          "4000",
				Vesting:         86400 * 1096,
				VestingStep:     86400,
			},
			amount: big.NewInt(4000),
			variants: []variant{
				{
					currentTime: 1573823965 + 86400*750, // 1 638 623 965
					expectation: &SchemaNextRelease{
						Amount:    "2",
						Timestamp: 1573823965 + 86400*751, // 1 638 710 365
					},
				},
				{
					currentTime: 1573823965 + 86400*850, // 1 647 263 965
					expectation: &SchemaNextRelease{
						Amount:    "5",
						Timestamp: 1573823965 + 86400*851, // 1 647 350 365
					},
				},
				{
					currentTime: 1573823965 + 86400*1090, // 1 667 999 965
					expectation: &SchemaNextRelease{
						Amount:    "29",
						Timestamp: 1573823965 + 86400*1091, // 1 668 086 365
					},
				},
				{
					currentTime: 1573823965 + 86400*1097,
				},
			},
		},
		{
			name: "no scaling, 1096 points",
			deposit: models.Deposit{
				HoldReleaseDate: 100,
				Amount:          "5000000000",
				Vesting:         10960,
				VestingStep:     10,
			},
			amount: big.NewInt(5000000000),
			variants: []variant{
				{
					currentTime: 10,
					expectation: &SchemaNextRelease{
						Amount:    "11280",
						Timestamp: 100,
					},
				},
				{
					currentTime: 20,
					expectation: &SchemaNextRelease{
						Amount:    "11280",
						Timestamp: 100,
					},
				},
				{
					currentTime: 90,
					expectation: &SchemaNextRelease{
						Amount:    "11280",
						Timestamp: 100,
					},
				},
				{
					currentTime: 100,
					expectation: &SchemaNextRelease{
						Amount:    "11365",
						Timestamp: 110,
					},
				},
				{
					currentTime: 109,
					expectation: &SchemaNextRelease{
						Amount:    "11365",
						Timestamp: 110,
					},
				},
				{
					currentTime: 110,
					expectation: &SchemaNextRelease{
						Amount:    "11449",
						Timestamp: 120,
					},
				},
				{
					currentTime: 119,
					expectation: &SchemaNextRelease{
						Amount:    "11449",
						Timestamp: 120,
					},
				},
				{
					currentTime: 9850,
					expectation: &SchemaNextRelease{
						Amount:    "15283088",
						Timestamp: 9860,
					},
				},
				{
					currentTime: 10350,
					expectation: &SchemaNextRelease{
						Amount:    "22113376",
						Timestamp: 10360,
					},
				},
				{
					currentTime: 11350,
				},
			},
		},
		{
			name: "4k in 7steps",
			deposit: models.Deposit{
				HoldReleaseDate: 100,
				Amount:          "4000",
				Vesting:         70,
				VestingStep:     10,
			},
			amount: big.NewInt(4000),
			variants: []variant{
				{
					currentTime: 90,
					expectation: &SchemaNextRelease{
						Amount:    "0",
						Timestamp: 100,
					},
				},
				{
					currentTime: 100,
					expectation: &SchemaNextRelease{
						Amount:    "2",
						Timestamp: 110,
					},
				},
				{
					currentTime: 110,
					expectation: &SchemaNextRelease{
						Amount:    "9",
						Timestamp: 120,
					},
				},
				{
					currentTime: 120,
					expectation: &SchemaNextRelease{
						Amount:    "26",
						Timestamp: 130,
					},
				},
				{
					currentTime: 130,
					expectation: &SchemaNextRelease{
						Amount:    "86",
						Timestamp: 140,
					},
				},
				{
					currentTime: 140,
					expectation: &SchemaNextRelease{
						Amount:    "271",
						Timestamp: 150,
					},
				},
				{
					currentTime: 150,
					expectation: &SchemaNextRelease{
						Amount:    "868",
						Timestamp: 160,
					},
				},
				{
					currentTime: 160,
					expectation: &SchemaNextRelease{
						Amount:    "2738",
						Timestamp: 170,
					},
				},
				{
					currentTime: 170,
				},
			},
		},
		{
			name: "no balance",
			deposit: models.Deposit{
				HoldReleaseDate: 100,
				Amount:          "4000",
				Vesting:         70,
				VestingStep:     10,
			},
			amount:  big.NewInt(4000),
			balance: big.NewInt(0),
			variants: []variant{
				{currentTime: 90, expectation: nil},
				{currentTime: 100, expectation: nil},
				{currentTime: 110, expectation: nil},
			},
		},
		{
			name: "not enough balance for next release",
			deposit: models.Deposit{
				HoldReleaseDate: 100,
				Amount:          "4000",
				Vesting:         70,
				VestingStep:     10,
			},
			amount:  big.NewInt(4000),
			balance: big.NewInt(16),
			variants: []variant{
				{
					currentTime: 100,
					expectation: &SchemaNextRelease{
						Amount:    "2",
						Timestamp: 110,
					},
				},
				{
					currentTime: 110,
					expectation: &SchemaNextRelease{
						Amount:    "9",
						Timestamp: 120,
					},
				},
				{
					currentTime: 120,
					expectation: &SchemaNextRelease{
						Amount:    "16",
						Timestamp: 130,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, e := range tc.variants {
				balance := tc.balance
				if balance == nil {
					balance = tc.amount
				}
				res := NextRelease(e.currentTime, tc.amount, balance, tc.deposit)
				assert.Equal(t, e.expectation, res)
			}
		})
	}
}

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
