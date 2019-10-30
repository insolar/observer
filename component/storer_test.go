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
	"github.com/insolar/observer/connectivity"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/models"
	"github.com/insolar/observer/observability"
)

func Test_makeStorer(t *testing.T) {
	cfg := configuration.Default()
	obs := observability.Make(cfg)
	conn := connectivity.Make(cfg, obs)
	storer := makeStorer(cfg, obs, conn)

	b := &beauty{
		transfers: []*observer.Transfer{{}},
	}
	s := &state{}

	cfg.DB.Attempts = 1
	cfg.DB.AttemptInterval = time.Nanosecond

	require.NotPanics(t, func() {
		storer(b, s)
	})
}

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
						TransactionID:       expectedTransactions[0].TransactionID,
						Type:                expectedTransactions[0].Type,
						PulseNumber:         expectedTransactions[0].PulseRecord[0],
						RecordNumber:        expectedTransactions[0].PulseRecord[1],
						MemberFromReference: expectedTransactions[0].MemberFromReference,
						MemberToReference:   expectedTransactions[0].MemberToReference,
						Amount:              expectedTransactions[0].Amount,
					},
					{
						TransactionID:      expectedTransactions[1].TransactionID,
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
						TransactionID: expectedTransactions[0].TransactionID,
						Fee:           expectedTransactions[0].Fee,
					},
					{
						TransactionID: expectedTransactions[1].TransactionID,
						Fee:           expectedTransactions[1].Fee,
					},
				})
			},
			func() error {
				return StoreTxSagaResult(tx, []observer.TxSagaResult{
					{
						TransactionID:      expectedTransactions[0].TransactionID,
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
