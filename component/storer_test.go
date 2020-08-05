// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package component

import (
	"bytes"
	"context"
	"math/rand"
	"sort"
	"strconv"
	"strings"
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
	amount := strconv.Itoa(rand.Int())
	expectedTransactions := []models.Transaction{
		{
			TransactionID:       gen.RecordReference().Bytes(),
			Type:                models.TTypeTransfer,
			PulseRecord:         [2]int64{rand.Int63(), rand.Int63()},
			MemberFromReference: gen.Reference().Bytes(),
			MemberToReference:   gen.Reference().Bytes(),
			Amount:              amount,
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
			Amount:             amount,
			Fee:                strconv.Itoa(rand.Int()),
			StatusRegistered:   true,
			StatusSent:         true,
			StatusFinished:     false,
		},
	}
	_ = db.RunInTransaction(func(tx *pg.Tx) error {
		err := StoreTxRegister(tx, []observer.TxRegister{
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
		require.NoError(t, err)

		// Create different update functions.
		funcs := []func() error{
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
			funcs[i], funcs[j] = funcs[j], funcs[i]
		})
		for _, f := range funcs {
			err := f()
			require.NoError(t, err)
		}

		// Select transactions from db.
		selected := make([]models.Transaction, 2)
		res, err := tx.Query(&selected, `SELECT * FROM simple_transactions WHERE amount = ? ORDER BY tx_id`, amount)
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

func TestPanicSimpleTransactionsWithResults(t *testing.T) {
	_ = db.RunInTransaction(func(tx *pg.Tx) error {
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

		err := StoreTxRegister(db, []observer.TxRegister{
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

		require.NoError(t, err)

		require.Panics(t, func() {
			_ = StoreTxResult(db, []observer.TxResult{
				{
					TransactionID: gen.Reference(), // This ID does not exists in db, so we should panic for this.
					Fee:           expectedTransactions[0].Fee,
				},
				{
					TransactionID: *insolar.NewReferenceFromBytes(expectedTransactions[0].TransactionID),
					Fee:           expectedTransactions[0].Fee,
				},
				{
					TransactionID: *insolar.NewReferenceFromBytes(expectedTransactions[1].TransactionID),
					Fee:           expectedTransactions[1].Fee,
				},
			})
		})

		return tx.Rollback()
	})
}

func TestPanicSimpleTransactionsWithSagaResults(t *testing.T) {
	_ = db.RunInTransaction(func(tx *pg.Tx) error {
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

		err := StoreTxRegister(db, []observer.TxRegister{
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

		require.NoError(t, err)

		require.Panics(t, func() {
			_ = StoreTxSagaResult(db, []observer.TxSagaResult{
				{
					TransactionID:      gen.Reference(), // This ID does not exists in db, so we should panic for this.
					FinishSuccess:      expectedTransactions[0].FinishSuccess,
					FinishPulseNumber:  expectedTransactions[0].FinishPulseRecord[0],
					FinishRecordNumber: expectedTransactions[0].FinishPulseRecord[1],
				},
				{
					TransactionID:      *insolar.NewReferenceFromBytes(expectedTransactions[0].TransactionID),
					FinishSuccess:      expectedTransactions[0].FinishSuccess,
					FinishPulseNumber:  expectedTransactions[0].FinishPulseRecord[0],
					FinishRecordNumber: expectedTransactions[0].FinishPulseRecord[1],
				},
			})
		})

		return tx.Rollback()
	})
}

func TestStoreSimpleDeposit(t *testing.T) {
	obs := observability.Make(context.Background())

	ref := gen.RecordReference()
	memberRef := gen.RecordReference()
	state := gen.RecordReference()
	transferDate := time.Now().Unix()
	holdDate := time.Now().Unix() + 5
	txHash := "tx_hash_0"
	expectedDeposit := []models.Deposit{
		{
			Reference:       ref.Bytes(),
			MemberReference: memberRef.Bytes(),
			EtheriumHash:    txHash,
			State:           state.GetLocal().Bytes(),
			Timestamp:       transferDate,
			HoldReleaseDate: holdDate,
			Amount:          "100500",
			Balance:         "100",
			Vesting:         10,
			VestingStep:     5,
			InnerStatus:     models.DepositStatusCreated,
			VestingType:     models.DepositTypeNonLinear,
		},
	}

	_ = db.RunInTransaction(func(tx *pg.Tx) error {

		deposits := postgres.NewDepositStorage(obs, tx)

		err := deposits.Insert(observer.Deposit{
			EthHash:         expectedDeposit[0].EtheriumHash,
			Ref:             *insolar.NewReferenceFromBytes(expectedDeposit[0].Reference),
			Member:          *insolar.NewReferenceFromBytes(expectedDeposit[0].MemberReference),
			Timestamp:       transferDate,
			HoldReleaseDate: holdDate,
			Amount:          expectedDeposit[0].Amount,
			Balance:         expectedDeposit[0].Balance,
			DepositState:    *insolar.NewIDFromBytes(expectedDeposit[0].State),
			Vesting:         expectedDeposit[0].Vesting,
			VestingStep:     expectedDeposit[0].VestingStep,
			VestingType:     expectedDeposit[0].VestingType,
			DepositNumber:   100,
		})
		if err != nil {
			return err
		}

		// Select deposit from db.
		selected := make([]models.Deposit, 1)
		res, err := tx.Query(&selected, `SELECT * FROM deposits where eth_hash = ?`, txHash)
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

func TestStoreSeveralDepositsWithDepositsNumbers(t *testing.T) {
	obs := observability.Make(context.Background())

	ref := gen.RecordReference()
	memberRef := gen.RecordReference()
	state := gen.RecordReference()
	transferDate := time.Now().Unix()
	holdDate := time.Now().Unix() + 5

	expectedDeposit := []models.Deposit{
		{
			Reference:       ref.Bytes(),
			MemberReference: memberRef.Bytes(),
			EtheriumHash:    "tx_hash_0",
			State:           state.GetLocal().Bytes(),
			HoldReleaseDate: holdDate,
			Amount:          "100500",
			Balance:         "100",
			Vesting:         10,
			VestingStep:     5,
			Timestamp:       transferDate,
			DepositNumber:   newInt(1),
			VestingType:     models.DepositTypeNonLinear,
		},
		{
			Reference:       ref.Bytes(),
			MemberReference: memberRef.Bytes(),
			EtheriumHash:    "tx_hash_1",
			State:           state.GetLocal().Bytes(),
			HoldReleaseDate: holdDate,
			Amount:          "100",
			Balance:         "10",
			Vesting:         10,
			VestingStep:     5,
			Timestamp:       transferDate,
			DepositNumber:   newInt(2),
			VestingType:     models.DepositTypeNonLinear,
		},
		{
			Reference:       gen.RecordReference().Bytes(),
			MemberReference: gen.RecordReference().Bytes(),
			EtheriumHash:    "tx_hash_2",
			State:           gen.RecordReference().GetLocal().Bytes(),
			HoldReleaseDate: holdDate,
			Amount:          "200500",
			Balance:         "200",
			Vesting:         10,
			VestingStep:     5,
			Timestamp:       transferDate,
			DepositNumber:   newInt(1),
			VestingType:     models.DepositTypeNonLinear,
		},
	}

	_ = db.RunInTransaction(func(tx *pg.Tx) error {

		deposits := postgres.NewDepositStorage(obs, tx)

		for _, dep := range expectedDeposit {
			err := deposits.Insert(observer.Deposit{
				EthHash:         dep.EtheriumHash,
				Ref:             *insolar.NewReferenceFromBytes(dep.Reference),
				Member:          *insolar.NewReferenceFromBytes(dep.MemberReference),
				Timestamp:       transferDate,
				HoldReleaseDate: holdDate,
				Amount:          dep.Amount,
				Balance:         dep.Balance,
				DepositState:    *insolar.NewIDFromBytes(dep.State),
				Vesting:         dep.Vesting,
				VestingStep:     dep.VestingStep,
				DepositNumber:   *dep.DepositNumber,
				VestingType:     models.DepositTypeNonLinear,
			})
			if err != nil {
				return err
			}
		}

		// Select deposit from db.
		selected := make([]models.Deposit, 3)
		res, err := tx.Query(&selected, `SELECT * FROM deposits`)
		require.NoError(t, err)
		require.Equal(t, 3, res.RowsReturned())

		// Reset ID field to simplify comparing.
		for i, t := range selected {
			selected[i] = t
		}

		// Compare deposits.
		assert.Equal(t, expectedDeposit, selected)

		return tx.Rollback()
	})
}

func TestStoreTxDepositData(t *testing.T) {
	require.NoError(t, db.RunInTransaction(func(tx *pg.Tx) error {
		expectedTransactions := []models.Transaction{
			{
				ID:                   1,
				TransactionID:        gen.RecordReference().Bytes(),
				Type:                 models.TTypeRelease,
				PulseRecord:          [2]int64{rand.Int63(), rand.Int63()},
				MemberFromReference:  gen.Reference().Bytes(),
				MemberToReference:    gen.Reference().Bytes(),
				DepositFromReference: gen.Reference().Bytes(),
				Amount:               strconv.Itoa(rand.Int()),
				StatusRegistered:     true,
			},
		}

		err := StoreTxRegister(tx, []observer.TxRegister{
			{
				TransactionID:       *insolar.NewReferenceFromBytes(expectedTransactions[0].TransactionID),
				Type:                expectedTransactions[0].Type,
				PulseNumber:         expectedTransactions[0].PulseRecord[0],
				RecordNumber:        expectedTransactions[0].PulseRecord[1],
				MemberFromReference: expectedTransactions[0].MemberFromReference,
				MemberToReference:   expectedTransactions[0].MemberToReference,
				Amount:              expectedTransactions[0].Amount,
			},
		})
		if err != nil {
			return err
		}
		err = StoreTxDepositData(tx, []observer.TxDepositTransferUpdate{
			{
				TransactionID:        *insolar.NewReferenceFromBytes(expectedTransactions[0].TransactionID),
				DepositFromReference: expectedTransactions[0].DepositFromReference,
			},
		})
		if err != nil {
			return err
		}
		res, err := GetTx(context.Background(), tx, expectedTransactions[0].TransactionID)
		if err != nil {
			return err
		}

		res.ID = 1

		require.Equal(t, expectedTransactions[0], *res)

		return nil
	}))
}

func TestStorerOK(t *testing.T) {
	cfg := configuration.Observer{}.Default()
	obs := observability.Make(context.Background())

	storer := makeStorer(cfg, obs, fakeConn{})

	stats := storer(&beauty{
		pulse: &observer.Pulse{
			Number: insolar.GenesisPulse.PulseNumber,
			Nodes: []insolar.Node{
				{
					ID:   gen.Reference(),
					Role: insolar.StaticRoleHeavyMaterial,
				},
			},
		},
		deposits: map[insolar.ID]observer.Deposit{
			gen.ID(): {
				EthHash:         strings.ToLower("0x5ca5e6417f818ba1c74d"),
				Ref:             gen.Reference(),
				Member:          gen.Reference(),
				Timestamp:       time.Now().Unix(),
				HoldReleaseDate: 0,
				Amount:          "120",
				Balance:         "123",
				DepositState:    gen.ID(),
				Vesting:         10,
				VestingStep:     10,
				VestingType:     models.DepositTypeNonLinear,
			},
		},
	}, &state{})

	assert.Equal(t, &observer.Statistic{
		Pulse: insolar.GenesisPulse.PulseNumber,
		Nodes: 1,
	}, stats)
}

func newInt(val int64) *int64 {
	return &val
}

func TestStoreFailedTransactions(t *testing.T) {
	expectedTransactions := []models.Transaction{
		{
			TransactionID:       gen.RecordReference().Bytes(),
			Type:                models.TTypeTransfer,
			PulseRecord:         [2]int64{rand.Int63(), rand.Int63()},
			MemberFromReference: gen.Reference().Bytes(),
			MemberToReference:   gen.Reference().Bytes(),
			Amount:              strconv.Itoa(rand.Int()),
			Fee:                 strconv.Itoa(rand.Int()),
			FinishSuccess:       false,
			FinishPulseRecord:   [2]int64{rand.Int63(), rand.Int63()},
			StatusRegistered:    true,
			StatusSent:          true,
			StatusFinished:      true,
		},
		{
			TransactionID:       gen.RecordReference().Bytes(),
			Type:                models.TTypeTransfer,
			PulseRecord:         [2]int64{rand.Int63(), rand.Int63()},
			MemberFromReference: gen.Reference().Bytes(),
			MemberToReference:   gen.Reference().Bytes(),
			Amount:              strconv.Itoa(rand.Int()),
			Fee:                 strconv.Itoa(rand.Int()),
			FinishSuccess:       false,
			FinishPulseRecord:   [2]int64{rand.Int63(), rand.Int63()},
			StatusRegistered:    true,
			StatusSent:          true,
			StatusFinished:      true,
		},
		{
			TransactionID:       gen.RecordReference().Bytes(),
			Type:                models.TTypeTransfer,
			PulseRecord:         [2]int64{rand.Int63(), rand.Int63()},
			MemberFromReference: gen.Reference().Bytes(),
			MemberToReference:   gen.Reference().Bytes(),
			Amount:              strconv.Itoa(rand.Int()),
			Fee:                 strconv.Itoa(rand.Int()),
			FinishSuccess:       false,
			FinishPulseRecord:   [2]int64{rand.Int63(), rand.Int63()},
			StatusRegistered:    true,
			StatusSent:          true,
			StatusFinished:      true,
		},
		{
			TransactionID:       gen.RecordReference().Bytes(),
			Type:                models.TTypeTransfer,
			PulseRecord:         [2]int64{rand.Int63(), rand.Int63()},
			MemberFromReference: gen.Reference().Bytes(),
			MemberToReference:   gen.Reference().Bytes(),
			Amount:              strconv.Itoa(rand.Int()),
			Fee:                 strconv.Itoa(rand.Int()),
			FinishSuccess:       false,
			FinishPulseRecord:   [2]int64{0, 0},
			StatusRegistered:    true,
			StatusSent:          true,
			StatusFinished:      false,
		},
	}
	_ = db.RunInTransaction(func(tx *pg.Tx) error {
		err := StoreTxRegister(tx, []observer.TxRegister{
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
				TransactionID:       *insolar.NewReferenceFromBytes(expectedTransactions[1].TransactionID),
				Type:                expectedTransactions[1].Type,
				PulseNumber:         expectedTransactions[1].PulseRecord[0],
				RecordNumber:        expectedTransactions[1].PulseRecord[1],
				MemberFromReference: expectedTransactions[1].MemberFromReference,
				MemberToReference:   expectedTransactions[1].MemberToReference,
				Amount:              expectedTransactions[1].Amount,
			},
			{
				TransactionID:       *insolar.NewReferenceFromBytes(expectedTransactions[2].TransactionID),
				Type:                expectedTransactions[2].Type,
				PulseNumber:         expectedTransactions[2].PulseRecord[0],
				RecordNumber:        expectedTransactions[2].PulseRecord[1],
				MemberFromReference: expectedTransactions[2].MemberFromReference,
				MemberToReference:   expectedTransactions[2].MemberToReference,
				Amount:              expectedTransactions[2].Amount,
			},
			{
				TransactionID:       *insolar.NewReferenceFromBytes(expectedTransactions[3].TransactionID),
				Type:                expectedTransactions[3].Type,
				PulseNumber:         expectedTransactions[3].PulseRecord[0],
				RecordNumber:        expectedTransactions[3].PulseRecord[1],
				MemberFromReference: expectedTransactions[3].MemberFromReference,
				MemberToReference:   expectedTransactions[3].MemberToReference,
				Amount:              expectedTransactions[3].Amount,
			},
		})
		require.NoError(t, err)

		t.Run("with saga", func(t *testing.T) {
			err = StoreTxSagaResult(tx, []observer.TxSagaResult{
				{
					TransactionID:      *insolar.NewReferenceFromBytes(expectedTransactions[0].TransactionID),
					FinishSuccess:      expectedTransactions[0].FinishSuccess,
					FinishPulseNumber:  expectedTransactions[0].FinishPulseRecord[0],
					FinishRecordNumber: expectedTransactions[0].FinishPulseRecord[1],
				},
			})
			require.NoError(t, err)

			err = StoreTxResult(tx, []observer.TxResult{
				{
					TransactionID: *insolar.NewReferenceFromBytes(expectedTransactions[0].TransactionID),
					Fee:           expectedTransactions[0].Fee,
					Failed: &observer.TxFailed{
						FinishPulseNumber:  expectedTransactions[0].FinishPulseRecord[0],
						FinishRecordNumber: expectedTransactions[0].FinishPulseRecord[1],
					},
				},
			})
			require.NoError(t, err)

			ctx := context.Background()

			txID := insolar.NewReferenceFromBytes(expectedTransactions[0].TransactionID)
			res, err := GetTx(ctx, tx, txID.Bytes())
			require.NoError(t, err)
			res.ID = 0
			assert.Equal(t, &expectedTransactions[0], res)
		})

		t.Run("without saga", func(t *testing.T) {
			err = StoreTxResult(tx, []observer.TxResult{
				{
					TransactionID: *insolar.NewReferenceFromBytes(expectedTransactions[1].TransactionID),
					Fee:           expectedTransactions[1].Fee,
					Failed: &observer.TxFailed{
						FinishPulseNumber:  expectedTransactions[1].FinishPulseRecord[0],
						FinishRecordNumber: expectedTransactions[1].FinishPulseRecord[1],
					},
				},
			})
			require.NoError(t, err)

			ctx := context.Background()

			txID := insolar.NewReferenceFromBytes(expectedTransactions[1].TransactionID)
			res, err := GetTx(ctx, tx, txID.Bytes())
			require.NoError(t, err)
			res.ID = 0
			assert.Equal(t, &expectedTransactions[1], res)
		})

		t.Run("success + fail", func(t *testing.T) {
			err = StoreTxResult(tx, []observer.TxResult{
				{
					TransactionID: *insolar.NewReferenceFromBytes(expectedTransactions[2].TransactionID),
					Fee:           expectedTransactions[2].Fee,
					Failed: &observer.TxFailed{
						FinishPulseNumber:  expectedTransactions[2].FinishPulseRecord[0],
						FinishRecordNumber: expectedTransactions[2].FinishPulseRecord[1],
					},
				},
				{
					TransactionID: *insolar.NewReferenceFromBytes(expectedTransactions[3].TransactionID),
					Fee:           expectedTransactions[3].Fee,
				},
			})
			require.NoError(t, err)

			ctx := context.Background()

			txID := insolar.NewReferenceFromBytes(expectedTransactions[2].TransactionID)
			res, err := GetTx(ctx, tx, txID.Bytes())
			require.NoError(t, err)
			res.ID = 0
			assert.Equal(t, &expectedTransactions[2], res)

			txID = insolar.NewReferenceFromBytes(expectedTransactions[3].TransactionID)
			res, err = GetTx(ctx, tx, txID.Bytes())
			require.NoError(t, err)
			res.ID = 0
			assert.Equal(t, &expectedTransactions[3], res)
		})

		return tx.Rollback()
	})
}
