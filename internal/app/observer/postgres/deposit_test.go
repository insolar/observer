// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres_test

import (
	"context"
	"github.com/insolar/observer/internal/testutils"
	"testing"
	"time"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
	"github.com/insolar/observer/observability"
)

func newInt(val int64) *int64 {
	return &val
}

func TestDepositStorage_Insert(t *testing.T) {
	defer testutils.TruncateTables(t, db, []interface{}{
		&models.Deposit{},
	})
	depositRepo := postgres.NewDepositStorage(observability.Make(context.Background()), db)

	t.Run("not confirmed", func(t *testing.T) {
		now := time.Now().Unix()

		deposit := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Member:          gen.Reference(),
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			VestingType:     models.DepositTypeDefaultFund,
			DepositState:    gen.ID(),
		}

		err := depositRepo.Insert(deposit)
		require.NoError(t, err, "insert")

		res, err := depositRepo.GetDeposit(deposit.Ref.Bytes())
		require.NoError(t, err, "get deposit")
		require.Equal(t, &models.Deposit{
			Reference:       deposit.Ref.Bytes(),
			MemberReference: deposit.Member.Bytes(),
			EtheriumHash:    deposit.EthHash,
			State:           deposit.DepositState.Bytes(),
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			Timestamp:       now,
			InnerStatus:     models.DepositStatusCreated,
			VestingType:     models.DepositTypeDefaultFund,
		}, res)
	})

	t.Run("confirmed", func(t *testing.T) {
		now := time.Now().Unix()

		deposit := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Member:          gen.Reference(),
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			DepositState:    gen.ID(),
			IsConfirmed:     true,
			VestingType:     models.DepositTypeDefaultFund,
		}

		err := depositRepo.Insert(deposit)
		require.NoError(t, err, "insert")

		res, err := depositRepo.GetDeposit(deposit.Ref.Bytes())
		require.NoError(t, err, "get deposit")
		require.Equal(t, &models.Deposit{
			Reference:       deposit.Ref.Bytes(),
			MemberReference: deposit.Member.Bytes(),
			EtheriumHash:    deposit.EthHash,
			State:           deposit.DepositState.Bytes(),
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			Timestamp:       now,
			InnerStatus:     models.DepositStatusConfirmed,
			VestingType:     deposit.VestingType,
			DepositNumber:   newInt(1),
		}, res)
	})
}

func TestDepositStorage_Update(t *testing.T) {
	defer testutils.TruncateTables(t, db, []interface{}{
		&models.Deposit{},
	})
	depositRepo := postgres.NewDepositStorage(observability.Make(context.Background()), db)

	t.Run("ok", func(t *testing.T) {
		now := time.Now().Unix()

		deposit := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Member:          gen.Reference(),
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			DepositState:    gen.ID(),
			VestingType:     models.DepositTypeDefaultFund,
		}

		err := depositRepo.Insert(deposit)
		require.NoError(t, err, "insert")

		update := observer.DepositUpdate{
			ID:              gen.ID(),
			HoldReleaseDate: now + 1,
			Timestamp:       now - 10,
			Amount:          "100",
			Balance:         "100",
			PrevState:       deposit.DepositState,
			TxHash:          "123",
			IsConfirmed:     true,
		}

		err = depositRepo.Update(update)
		require.NoError(t, err, "update")

		res, err := depositRepo.GetDeposit(deposit.Ref.Bytes())
		require.NoError(t, err, "get deposit")
		require.Equal(t, &models.Deposit{
			Reference:       deposit.Ref.Bytes(),
			MemberReference: deposit.Member.Bytes(),
			EtheriumHash:    deposit.EthHash,
			State:           update.ID.Bytes(),
			HoldReleaseDate: now + 1,
			Amount:          "100",
			Balance:         "100",
			Timestamp:       now - 10,
			DepositNumber:   newInt(1),
			InnerStatus:     models.DepositStatusConfirmed,
			VestingType:     deposit.VestingType,
		}, res)
	})

	t.Run("two deposits", func(t *testing.T) {
		now := time.Now().Unix()

		deposit1 := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Member:          gen.Reference(),
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			DepositState:    gen.ID(),
			VestingType:     models.DepositTypeLinear,
		}

		err := depositRepo.Insert(deposit1)
		require.NoError(t, err, "insert")

		update1 := observer.DepositUpdate{
			ID:              gen.ID(),
			HoldReleaseDate: now + 1,
			Timestamp:       now,
			Amount:          "100",
			Balance:         "100",
			PrevState:       deposit1.DepositState,
			TxHash:          "123",
			IsConfirmed:     true,
		}

		err = depositRepo.Update(update1)
		require.NoError(t, err, "update")

		res, err := depositRepo.GetDeposit(deposit1.Ref.Bytes())
		require.NoError(t, err, "get deposit")
		require.Equal(t, &models.Deposit{
			Reference:       deposit1.Ref.Bytes(),
			MemberReference: deposit1.Member.Bytes(),
			EtheriumHash:    deposit1.EthHash,
			State:           update1.ID.Bytes(),
			HoldReleaseDate: now + 1,
			Amount:          "100",
			Balance:         "100",
			Timestamp:       now,
			DepositNumber:   newInt(1),
			InnerStatus:     models.DepositStatusConfirmed,
			VestingType:     deposit1.VestingType,
		}, res)

		deposit2 := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Member:          deposit1.Member,
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			DepositState:    gen.ID(),
			VestingType:     models.DepositTypeLinear,
		}

		err = depositRepo.Insert(deposit2)
		require.NoError(t, err, "insert")

		update2 := observer.DepositUpdate{
			ID:              gen.ID(),
			HoldReleaseDate: now + 1,
			Timestamp:       now,
			Amount:          "100",
			Balance:         "100",
			PrevState:       deposit2.DepositState,
			TxHash:          "123",
			IsConfirmed:     true,
		}

		err = depositRepo.Update(update2)
		require.NoError(t, err, "update")

		res, err = depositRepo.GetDeposit(deposit2.Ref.Bytes())
		require.NoError(t, err, "get deposit")
		require.Equal(t, &models.Deposit{
			Reference:       deposit2.Ref.Bytes(),
			MemberReference: deposit2.Member.Bytes(),
			EtheriumHash:    deposit2.EthHash,
			State:           update2.ID.Bytes(),
			HoldReleaseDate: now + 1,
			Amount:          "100",
			Balance:         "100",
			Timestamp:       now,
			DepositNumber:   newInt(2),
			InnerStatus:     models.DepositStatusConfirmed,
			VestingType:     deposit2.VestingType,
		}, res)
	})

	t.Run("two deposits per user", func(t *testing.T) {

		err := db.RunInTransaction(func(tx *pg.Tx) error {
			now := time.Now().Unix()

			depositRepo := postgres.NewDepositStorage(observability.Make(context.Background()), db)

			memberRef := gen.Reference()
			stateID := gen.ID()

			deposit1 := observer.Deposit{
				EthHash:         "123",
				Ref:             gen.Reference(),
				IsConfirmed:     true,
				Timestamp:       now,
				HoldReleaseDate: now,
				Amount:          "100",
				Balance:         "100",
				DepositState:    stateID,
				VestingType:     models.DepositTypeNonLinear,
			}

			err := depositRepo.Insert(deposit1)
			require.NoError(t, err, "insert")

			res, err := depositRepo.GetDeposit(deposit1.Ref.Bytes())
			require.NoError(t, err, "get deposit")
			require.Equal(t, &models.Deposit{
				Reference:       deposit1.Ref.Bytes(),
				EtheriumHash:    deposit1.EthHash,
				State:           stateID.Bytes(),
				HoldReleaseDate: now,
				Amount:          "100",
				Balance:         "100",
				Timestamp:       now,
				InnerStatus:     models.DepositStatusConfirmed,
				VestingType:     models.DepositTypeNonLinear,
			}, res)

			deposit2 := observer.Deposit{
				EthHash:         "123",
				Ref:             gen.Reference(),
				Timestamp:       now,
				HoldReleaseDate: now,
				Amount:          "100",
				Balance:         "100",
				DepositState:    gen.ID(),
				IsConfirmed:     true,
				VestingType:     models.DepositTypeNonLinear,
			}

			err = depositRepo.Insert(deposit2)
			require.NoError(t, err, "insert")

			res, err = depositRepo.GetDeposit(deposit2.Ref.Bytes())
			require.NoError(t, err, "get deposit")
			require.Equal(t, &models.Deposit{
				Reference:       deposit2.Ref.Bytes(),
				EtheriumHash:    deposit2.EthHash,
				State:           deposit2.DepositState.Bytes(),
				HoldReleaseDate: now,
				Amount:          "100",
				Balance:         "100",
				Timestamp:       now,
				InnerStatus:     models.DepositStatusConfirmed,
				VestingType:     models.DepositTypeNonLinear,
			}, res)

			err = depositRepo.SetMember(deposit1.Ref, memberRef)
			require.NoError(t, err)
			err = depositRepo.SetMember(deposit2.Ref, memberRef)
			require.NoError(t, err)

			res, err = depositRepo.GetDeposit(deposit1.Ref.Bytes())
			require.NoError(t, err, "get deposit")
			require.Equal(t, &models.Deposit{
				Reference:       deposit1.Ref.Bytes(),
				MemberReference: memberRef.Bytes(),
				EtheriumHash:    deposit1.EthHash,
				State:           stateID.Bytes(),
				HoldReleaseDate: now,
				Amount:          "100",
				Balance:         "100",
				Timestamp:       now,
				DepositNumber:   newInt(1),
				InnerStatus:     models.DepositStatusConfirmed,
				VestingType:     models.DepositTypeNonLinear,
			}, res)

			res, err = depositRepo.GetDeposit(deposit2.Ref.Bytes())
			require.NoError(t, err, "get deposit")
			require.Equal(t, &models.Deposit{
				Reference:       deposit2.Ref.Bytes(),
				MemberReference: memberRef.Bytes(),
				EtheriumHash:    deposit2.EthHash,
				State:           deposit2.DepositState.Bytes(),
				HoldReleaseDate: now,
				Amount:          "100",
				Balance:         "100",
				Timestamp:       now,
				DepositNumber:   newInt(2),
				InnerStatus:     models.DepositStatusConfirmed,
				VestingType:     models.DepositTypeNonLinear,
			}, res)

			return nil
		})

		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		update := observer.DepositUpdate{
			ID:              gen.ID(),
			HoldReleaseDate: time.Now().Unix() + 1,
			Amount:          "100",
			Balance:         "100",
			PrevState:       gen.ID(),
			TxHash:          "123",
			IsConfirmed:     true,
		}

		err := depositRepo.Update(update)
		require.Error(t, err, "update")
	})

	t.Run("failed to update", func(t *testing.T) {
		now := time.Now().Unix()

		deposit := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Member:          gen.Reference(),
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			DepositState:    gen.ID(),
			VestingType:     models.DepositTypeNonLinear,
		}

		err := depositRepo.Insert(deposit)
		require.NoError(t, err, "insert")

		update := observer.DepositUpdate{
			ID:              gen.ID(),
			HoldReleaseDate: now + 1,
			Amount:          "100",
			Balance:         "0",
			PrevState:       gen.ID(),
			TxHash:          "123",
			IsConfirmed:     true,
		}

		err = depositRepo.Update(update)
		require.Error(t, err, "update")
	})
}

func TestMemberSet(t *testing.T) {
	defer testutils.TruncateTables(t, db, []interface{}{
		&models.Deposit{},
	})
	depositRepo := postgres.NewDepositStorage(observability.Make(context.Background()), db)

	t.Run("ok", func(t *testing.T) {
		now := time.Now().Unix()

		deposit := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			DepositState:    gen.ID(),
			VestingType:     models.DepositTypeNonLinear,
		}

		memberRef := gen.Reference()

		err := depositRepo.Insert(deposit)
		require.NoError(t, err, "insert")

		err = depositRepo.SetMember(deposit.Ref, memberRef)
		require.NoError(t, err, "SetMember")

		res, err := depositRepo.GetDeposit(deposit.Ref.Bytes())
		require.NoError(t, err, "get deposit")
		require.Equal(t, &models.Deposit{
			Reference:       deposit.Ref.Bytes(),
			MemberReference: memberRef.Bytes(),
			EtheriumHash:    deposit.EthHash,
			State:           deposit.DepositState.Bytes(),
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			Timestamp:       now,
			InnerStatus:     models.DepositStatusCreated,
			VestingType:     models.DepositTypeNonLinear,
		}, res)
	})

	t.Run("updated before", func(t *testing.T) {
		now := time.Now().Unix()

		deposit := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			DepositState:    gen.ID(),
			VestingType:     models.DepositTypeLinear,
		}

		memberRef := gen.Reference()

		err := depositRepo.Insert(deposit)
		require.NoError(t, err, "insert")

		newState := gen.ID()

		update := observer.DepositUpdate{
			ID:              newState,
			HoldReleaseDate: now + 1,
			Amount:          "100",
			Balance:         "20",
			PrevState:       deposit.DepositState,
			TxHash:          "123",
			IsConfirmed:     true,
		}

		err = depositRepo.Update(update)
		require.NoError(t, err, "update")

		err = depositRepo.SetMember(deposit.Ref, memberRef)
		require.NoError(t, err, "SetMember")

		res, err := depositRepo.GetDeposit(deposit.Ref.Bytes())
		require.NoError(t, err, "get deposit")
		require.Equal(t, &models.Deposit{
			Reference:       deposit.Ref.Bytes(),
			MemberReference: memberRef.Bytes(),
			EtheriumHash:    deposit.EthHash,
			State:           newState.Bytes(),
			HoldReleaseDate: now + 1,
			Amount:          "100",
			Balance:         "20",
			Timestamp:       now,
			InnerStatus:     models.DepositStatusConfirmed,
			VestingType:     models.DepositTypeLinear,
			DepositNumber:   newInt(1),
		}, res)
	})

	t.Run("updated after", func(t *testing.T) {
		now := time.Now().Unix()

		deposit := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			DepositState:    gen.ID(),
			VestingType:     models.DepositTypeLinear,
		}

		memberRef := gen.Reference()

		err := depositRepo.Insert(deposit)
		require.NoError(t, err, "insert")

		err = depositRepo.SetMember(deposit.Ref, memberRef)
		require.NoError(t, err, "SetMember")

		newState := gen.ID()

		update := observer.DepositUpdate{
			ID:              newState,
			HoldReleaseDate: now + 1,
			Amount:          "100",
			Balance:         "100",
			PrevState:       deposit.DepositState,
			TxHash:          "123",
			IsConfirmed:     true,
		}

		err = depositRepo.Update(update)
		require.NoError(t, err, "update")

		res, err := depositRepo.GetDeposit(deposit.Ref.Bytes())
		require.NoError(t, err, "get deposit")
		require.Equal(t, &models.Deposit{
			Reference:       deposit.Ref.Bytes(),
			MemberReference: memberRef.Bytes(),
			EtheriumHash:    deposit.EthHash,
			State:           newState.Bytes(),
			HoldReleaseDate: now + 1,
			Amount:          "100",
			Balance:         "100",
			Timestamp:       now,
			InnerStatus:     models.DepositStatusConfirmed,
			VestingType:     models.DepositTypeLinear,
			DepositNumber:   newInt(1),
		}, res)
	})

	t.Run("member already set", func(t *testing.T) {
		now := time.Now().Unix()

		deposit := observer.Deposit{
			EthHash:         "123",
			Ref:             gen.Reference(),
			Timestamp:       now,
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			DepositState:    gen.ID(),
			VestingType:     models.DepositTypeNonLinear,
		}

		err := depositRepo.Insert(deposit)
		require.NoError(t, err, "insert")

		memberRef := gen.Reference()

		err = depositRepo.SetMember(deposit.Ref, memberRef)
		require.NoError(t, err, "SetMember")

		newMemberRef := gen.Reference()

		err = depositRepo.SetMember(deposit.Ref, newMemberRef)
		require.Error(t, err, "SetMember")
		require.Contains(t, err.Error(), "Trying to update member for deposit that already has different member")

		res, err := depositRepo.GetDeposit(deposit.Ref.Bytes())
		require.NoError(t, err, "get deposit")
		require.Equal(t, &models.Deposit{
			Reference:       deposit.Ref.Bytes(),
			MemberReference: memberRef.Bytes(),
			EtheriumHash:    deposit.EthHash,
			State:           deposit.DepositState.Bytes(),
			HoldReleaseDate: now,
			Amount:          "100",
			Balance:         "0",
			Timestamp:       now,
			InnerStatus:     models.DepositStatusCreated,
			VestingType:     models.DepositTypeNonLinear,
		}, res)
	})

	t.Run("lost deposit", func(t *testing.T) {
		err := depositRepo.SetMember(gen.Reference(), gen.Reference())
		require.Error(t, err, "SetMember")
	})
}
