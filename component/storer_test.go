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
// +build integration

package component

import (
	"bytes"
	"context"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
	"github.com/insolar/observer/observability"
)

func TestStoreSimpleTransactions(t *testing.T) {
	expectedTransactions := []models.Transaction{
		{
			TransactionID:       gen.RecordReference().Bytes(),
			Type:                models.TTypeTransfer,
			PulseRecord:         [2]int64{rand.Int63(), rand.Int63()},
			MemberFromReference: gen.Reference().Bytes(),
			MemberToReference:   gen.Reference().Bytes(),
			Amount:              strconv.Itoa(rand.Int()),
			Fee:                 strconv.Itoa(rand.Int()),
			FinishSuccess:       rand.Int()/2 == 0,
			FinishPulseRecord:   [2]int64{rand.Int63(), rand.Int63()},
			StatusRegistered:    true,
			StatusSent:          true,
			StatusFinished:      true,
		},
		{
			TransactionID:      gen.RecordReference().Bytes(),
			Type:               models.TTypeMigration,
			PulseRecord:        [2]int64{rand.Int63(), rand.Int63()},
			DepositToReference: gen.Reference().Bytes(),
			Amount:             strconv.Itoa(rand.Int()),
			Fee:                strconv.Itoa(rand.Int()),
			StatusRegistered:   true,
			StatusSent:         true,
			StatusFinished:     false,
		},
	}
	_ = db.RunInTransaction(func(tx *pg.Tx) error {
		// Create different update functions.
		funcs := []func() error{
			func() error {
				return StoreTxRegister(tx, []observer.TxRegister{
					{
						TransactionID:       *insolar.NewReferenceFromBytes(expectedTransactions[0].TransactionID),
						Type:                expectedTransactions[0].Type,
						PulseNumber:         expectedTransactions[0].PulseRecord[0],
						RecordNumber:        expectedTransactions[0].PulseRecord[1],
						MemberFromReference: expectedTransactions[0].MemberFromReference,
						MemberToReference:   expectedTransactions[0].MemberToReference,
						Amount:              expectedTransactions[0].Amount,
					},
					{
						TransactionID:      *insolar.NewReferenceFromBytes(expectedTransactions[1].TransactionID),
						Type:               expectedTransactions[1].Type,
						PulseNumber:        expectedTransactions[1].PulseRecord[0],
						RecordNumber:       expectedTransactions[1].PulseRecord[1],
						DepositToReference: expectedTransactions[1].DepositToReference,
						Amount:             expectedTransactions[1].Amount,
					},
				})
			},
			func() error {
				return StoreTxResult(tx, []observer.TxResult{
					{
						TransactionID: *insolar.NewReferenceFromBytes(expectedTransactions[0].TransactionID),
						Fee:           expectedTransactions[0].Fee,
					},
					{
						TransactionID: *insolar.NewReferenceFromBytes(expectedTransactions[1].TransactionID),
						Fee:           expectedTransactions[1].Fee,
					},
				})
			},
			func() error {
				return StoreTxSagaResult(tx, []observer.TxSagaResult{
					{
						TransactionID:      *insolar.NewReferenceFromBytes(expectedTransactions[0].TransactionID),
						FinishSuccess:      expectedTransactions[0].FinishSuccess,
						FinishPulseNumber:  expectedTransactions[0].FinishPulseRecord[0],
						FinishRecordNumber: expectedTransactions[0].FinishPulseRecord[1],
					},
				})
			},
		}

		// Run functions in random order.
		rand.Shuffle(len(funcs), func(i, j int) {
			t := funcs[i]
			funcs[i] = funcs[j]
			funcs[j] = t
		})
		for _, f := range funcs {
			err := f()
			require.NoError(t, err)
		}

		// Select transactions from db.
		selected := make([]models.Transaction, 2)
		res, err := tx.Query(&selected, `SELECT * FROM simple_transactions ORDER BY tx_id`)
		require.NoError(t, err)
		require.Equal(t, 2, res.RowsReturned())
		// Reset ID field to simplify comparing.
		for i, t := range selected {
			t.ID = 0
			selected[i] = t
		}
		// Sort expected slice.
		sort.Slice(expectedTransactions, func(i, j int) bool {
			return bytes.Compare(expectedTransactions[i].TransactionID, expectedTransactions[j].TransactionID) == -1
		})
		// Compare transactions.
		assert.Equal(t, expectedTransactions, selected)

		ctx := context.Background()

		for i := range expectedTransactions {
			txID := insolar.NewReferenceFromBytes(expectedTransactions[i].TransactionID)
			res, err := GetTx(ctx, tx, txID.Bytes())
			require.NoError(t, err)
			res.ID = 0
			assert.Equal(t, &expectedTransactions[i], res)
		}
		return tx.Rollback()
	})
}

func TestStoreSimpleDeposit(t *testing.T) {
	cfg := configuration.Default()
	obs := observability.Make(cfg)

	ref := gen.RecordReference()
	memberRef := gen.RecordReference()
	state := gen.RecordReference()
	transferDate := time.Now().Unix()
	holdDate := time.Now().Unix() + 5

	expectedDeposit := []models.Deposit{
		{
			ID:              1,
			Reference:       ref.GetLocal().Bytes(),
			MemberReference: memberRef.GetLocal().Bytes(),
			EtheriumHash:    "tx_hash_0",
			State:           state.GetLocal().Bytes(),
			HoldReleaseDate: holdDate,
			Amount:          "100500",
			Balance:         "100",
			Vesting:         10,
			VestingStep:     5,
			TransferDate:    transferDate,
		},
	}

	_ = db.RunInTransaction(func(tx *pg.Tx) error {

		deposits := postgres.NewDepositStorage(obs, tx)

		err := deposits.Insert(&observer.Deposit{
			EthHash:         expectedDeposit[0].EtheriumHash,
			Ref:             *insolar.NewIDFromBytes(expectedDeposit[0].Reference),
			Member:          *insolar.NewIDFromBytes(expectedDeposit[0].MemberReference),
			Timestamp:       transferDate,
			HoldReleaseDate: holdDate,
			Amount:          expectedDeposit[0].Amount,
			Balance:         expectedDeposit[0].Balance,
			DepositState:    *insolar.NewIDFromBytes(expectedDeposit[0].State),
			Vesting:         expectedDeposit[0].Vesting,
			VestingStep:     expectedDeposit[0].VestingStep,
		})
		if err != nil {
			return err
		}

		// Select deposit from db.
		selected := make([]models.Deposit, 1)
		res, err := tx.Query(&selected, `SELECT * FROM deposits`)
		require.NoError(t, err)
		require.Equal(t, 1, res.RowsReturned())

		for i, t := range selected {
			selected[i] = t
		}

		// Compare deposits.
		assert.Equal(t, expectedDeposit, selected)

		return tx.Rollback()
	})
}
