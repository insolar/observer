// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
)

func TestNetworkStats(t *testing.T) {
	log := inslogger.FromContext(context.Background())
	pulseRepo := postgres.NewPulseStorage(log, db)

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
	require.Equal(t, models.NetworkStats{
		Created:           now,
		PulseNumber:       int(insolar.GenesisPulse.PulseNumber + 3),
		TotalTransactions: 2,
		MonthTransactions: 1,
		Nodes:             2,
		MaxTPS:            1,
		CurrentTPS:        1,
	}, res)
}
