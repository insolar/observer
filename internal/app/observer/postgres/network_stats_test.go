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

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
	"github.com/insolar/observer/observability"
)

func TestCountStats(t *testing.T) {
	cfg := &configuration.Configuration{
		DB: configuration.DB{
			Attempts: 1,
		},
		LogLevel: "debug",
	}

	pulseRepo := postgres.NewPulseStorage(cfg, observability.Make(cfg), db)

	now := time.Now()

	monthAgo := now.Add(-1*time.Hour*24*30 - 1)

	err := pulseRepo.Insert(&observer.Pulse{
		Number:    insolar.GenesisPulse.PulseNumber + 1,
		Entropy:   [64]byte{1, 2, 3},
		Timestamp: monthAgo.Unix(),
		Nodes:     []insolar.Node{{}},
	})
	require.NoError(t, err)

	err = pulseRepo.Insert(&observer.Pulse{
		Number:    insolar.GenesisPulse.PulseNumber + 2,
		Entropy:   [64]byte{1, 2, 3},
		Timestamp: now.Add(-1 * time.Hour).Unix(),
		Nodes:     []insolar.Node{{}},
	})
	require.NoError(t, err)

	err = pulseRepo.Insert(&observer.Pulse{
		Number:    insolar.GenesisPulse.PulseNumber + 3,
		Entropy:   [64]byte{1, 2, 3},
		Timestamp: now.Unix(),
		Nodes:     []insolar.Node{{}, {}},
	})
	require.NoError(t, err)

	txID1, txID2 := gen.Reference(), gen.Reference()

	err = component.StoreTxRegister(db, []observer.TxRegister{
		{
			TransactionID: txID1,
			Type:          models.TTypeTransfer,
			PulseNumber:   int64(insolar.GenesisPulse.PulseNumber),
		},
		{
			TransactionID: txID2,
			Type:          models.TTypeTransfer,
			PulseNumber:   int64(insolar.GenesisPulse.PulseNumber + 3),
		},
	})
	require.NoError(t, err)

	err = component.StoreTxSagaResult(db, []observer.TxSagaResult{
		{
			TransactionID:     txID1,
			FinishPulseNumber: int64(insolar.GenesisPulse.PulseNumber),
		},
		{
			TransactionID:     txID2,
			FinishPulseNumber: int64(insolar.GenesisPulse.PulseNumber + 3),
		},
	})

	repo := postgres.NewNetworkStatsRepository(db)
	res, err := repo.CountStats()
	require.NoError(t, err)
	res.Created = now
	require.Equal(t, postgres.NetworkStatsModel{
		Created:           now,
		PulseNumber:       int(insolar.GenesisPulse.PulseNumber + 3),
		TotalTransactions: 2,
		MonthTransactions: 1,
		Nodes:             2,
		MaxTPS:            1,
		CurrentTPS:        1,
	}, res)
}
