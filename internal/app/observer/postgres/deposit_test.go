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

package postgres_test

import (
	"context"
	"testing"
	"time"

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
			DepositState:    gen.ID(),
		}

		err := depositRepo.Insert(deposit)
		require.NoError(t, err, "insert")
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
		}

		err := depositRepo.Insert(deposit)
		require.NoError(t, err, "insert")
	})
}

func TestDepositStorage_Update(t *testing.T) {
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
		}

		err := depositRepo.Insert(deposit)
		require.NoError(t, err, "insert")

		update := observer.DepositUpdate{
			ID:              gen.ID(),
			HoldReleaseDate: now + 1,
			Timestamp:       now,
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
			Timestamp:       now,
			DepositNumber:   newInt(1),
			InnerStatus:     models.DepositStatusConfirmed,
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
		}, res)
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
