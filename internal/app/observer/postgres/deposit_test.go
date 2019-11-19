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
	"testing"
	"time"

	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/observability"
)

func TestDepositStorage_Update(t *testing.T) {
	cfg := &configuration.Configuration{
		DB: configuration.DB{
			Attempts: 1,
		},
		LogLevel: "debug",
	}

	depositRepo := postgres.NewDepositStorage(observability.Make(cfg), db)

	t.Run("ok", func(t *testing.T) {
		now := time.Now().Unix()

		deposit := &observer.Deposit{
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

		update := &observer.DepositUpdate{
			ID:              gen.ID(),
			HoldReleaseDate: now + 1,
			Amount:          "100",
			Balance:         "100",
			PrevState:       deposit.DepositState,
			TxHash:          "123",
			IsConfirmed:     true,
		}

		err = depositRepo.Update(update)
		require.NoError(t, err, "update")
	})

	t.Run("not found", func(t *testing.T) {
		update := &observer.DepositUpdate{
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

		deposit := &observer.Deposit{
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

		update := &observer.DepositUpdate{
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
