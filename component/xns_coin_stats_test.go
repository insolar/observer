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

package component

import (
	"testing"
	"time"

	"github.com/gojuno/minimock"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/labstack/gommon/random"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer/postgres"
)

func TestStatsManager_Coins(t *testing.T) {
	t.Parallel()
	mc := minimock.NewController(t)
	log := logrus.New()

	t.Run("small counts", func(t *testing.T) {
		t.Parallel()
		repo := postgres.NewStatsRepoMock(mc)
		repo.LastStatsMock.Return(postgres.StatsModel{
			Created:     time.Time{},
			Total:       "1000",
			Max:         "2000",
			Circulating: "100",
		}, nil)

		sm := NewStatsManager(log, repo)
		res, err := sm.Coins()
		require.NoError(t, err)
		require.Equal(t, "1000", res.Total)
		require.Equal(t, "2000", res.Max)
		require.Equal(t, "100", res.Circulating)
	})

	t.Run("medium counts", func(t *testing.T) {
		t.Parallel()
		repo := postgres.NewStatsRepoMock(mc)
		repo.LastStatsMock.Return(postgres.StatsModel{
			Created:     time.Time{},
			Total:       "111122220000000",
			Max:         "333331111222222",
			Circulating: "444444111111111",
		}, nil)

		sm := NewStatsManager(log, repo)
		res, err := sm.Coins()
		require.NoError(t, err)
		require.Equal(t, "111122220000000", res.Total)
		require.Equal(t, "333331111222222", res.Max)
		require.Equal(t, "444444111111111", res.Circulating)
	})

	t.Run("big counts", func(t *testing.T) {
		t.Parallel()
		repo := postgres.NewStatsRepoMock(mc)
		repo.LastStatsMock.Return(postgres.StatsModel{
			Created:     time.Time{},
			Total:       "111112222000000055555555",
			Max:         "333333111122222244444444",
			Circulating: "444444411111111199999999",
		}, nil)

		sm := NewStatsManager(log, repo)
		res, err := sm.Coins()
		require.NoError(t, err)
		require.Equal(t, "111112222000000055555555", res.Total)
		require.Equal(t, "333333111122222244444444", res.Max)
		require.Equal(t, "444444411111111199999999", res.Circulating)
	})
}

func TestStatsManager_CLI_command(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	member := gen.Reference()
	size := 10
	now := time.Now().Unix()

	for i := 0; i < size; i++ {
		memberModel := &postgres.MemberSchema{
			MemberRef:        gen.Reference().Bytes(),
			Balance:          "100",
			MigrationAddress: random.String(10),
			WalletRef:        gen.Reference().Bytes(),
			AccountState:     gen.ID().Bytes(),
			Status:           "TEST",
			AccountRef:       gen.Reference().Bytes(),
		}
		err := db.Insert(memberModel)
		require.NoError(t, err)
	}

	for i := 0; i < size; i++ {
		depModel := &postgres.DepositSchema{
			EthHash:         random.String(5),
			MemberRef:       member.Bytes(),
			DepositRef:      gen.Reference().Bytes(),
			TransferDate:    now,
			HoldReleaseDate: now + 10000,
			Amount:          "10000",
			Balance:         "10000",
			DepositState:    gen.ID().Bytes(),
			Vesting:         1000,
			VestingStep:     10,
		}
		err := db.Insert(depModel)
		require.NoError(t, err)
	}
	res, err := db.Model(&postgres.MemberSchema{}).Where("status=?", "TEST").Count()
	require.NoError(t, err)
	require.Equal(t, size, res)

	res, err = db.Model(&postgres.DepositSchema{}).Where("member_ref=?", member.Bytes()).Count()
	require.NoError(t, err)
	require.Equal(t, size, res)

	repo := postgres.NewStatsRepository(db)
	sr := NewStatsManager(log, repo)

	command := NewCalculateStatsCommand(log, db, sr)
	err = command.Run(nil)
	require.NoError(t, err)

	stats := &postgres.StatsModel{}
	err = db.Model(stats).Last()
	require.NoError(t, err)
	// todo check formula here
}
