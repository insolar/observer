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
	"github.com/insolar/observer/internal/models"
)

func TestStatsManager_Coins(t *testing.T) {
	t.Parallel()
	mc := minimock.NewController(t)
	log := logrus.New()

	t.Run("small counts", func(t *testing.T) {
		t.Parallel()
		repo := postgres.NewSupplyStatsRepoMock(mc)
		repo.LastStatsMock.Return(postgres.SupplyStatsModel{
			Created:     time.Time{},
			Total:       "1000",
			Max:         "2000",
			Circulating: "100",
		}, nil)

		sm := NewStatsManager(log, repo)
		res, err := sm.Supply()
		require.NoError(t, err)
		require.Equal(t, "1000", res.Total())
		require.Equal(t, "2000", res.Max())
		require.Equal(t, "100", res.Circulating())
	})

	t.Run("medium counts", func(t *testing.T) {
		t.Parallel()
		repo := postgres.NewSupplyStatsRepoMock(mc)
		repo.LastStatsMock.Return(postgres.SupplyStatsModel{
			Created:     time.Time{},
			Total:       "111122220000000",
			Max:         "333331111222222",
			Circulating: "444444111111111",
		}, nil)

		sm := NewStatsManager(log, repo)
		res, err := sm.Supply()
		require.NoError(t, err)
		require.Equal(t, "111122220000000", res.Total())
		require.Equal(t, "333331111222222", res.Max())
		require.Equal(t, "444444111111111", res.Circulating())
	})

	t.Run("big counts", func(t *testing.T) {
		t.Parallel()
		repo := postgres.NewSupplyStatsRepoMock(mc)
		repo.LastStatsMock.Return(postgres.SupplyStatsModel{
			Created:     time.Time{},
			Total:       "111112222000000055555555",
			Max:         "333333111122222244444444",
			Circulating: "444444411111111199999999",
		}, nil)

		sm := NewStatsManager(log, repo)
		res, err := sm.Supply()
		require.NoError(t, err)
		require.Equal(t, "111112222000000055555555", res.Total())
		require.Equal(t, "333333111122222244444444", res.Max())
		require.Equal(t, "444444411111111199999999", res.Circulating())
	})
}

// THIS TESTS ARE ORDER DEPENDENT!
func TestStatsManager_CLI_command(t *testing.T) {
	log := logrus.New()
	repo := postgres.NewSupplyStatsRepository(db)
	sr := NewStatsManager(log, repo)

	size := 10
	now := time.Now()

	getStats := func(dt *time.Time) XnsCoinStats {
		if dt != nil {
			log.Infoln("dt: ", dt.String())
		}
		command := NewCalculateStatsCommand(log, db, sr)
		stats, err := command.Run(dt)
		require.NoError(t, err)

		log.Infof("%+v", stats)
		return stats
	}

	_, err := db.Exec("truncate table members")
	require.NoError(t, err)
	_, err = db.Exec("truncate table deposits")
	require.NoError(t, err)

	initialStats := getStats(nil)
	require.Equal(t, "0", initialStats.Circulating())
	require.Equal(t, "0", initialStats.Total())
	require.Equal(t, "0", initialStats.Max())

	for i := 0; i < size; i++ {
		memberModel := &postgres.MemberSchema{
			MemberRef:        gen.Reference().Bytes(),
			Balance:          "0",
			MigrationAddress: random.String(10),
			WalletRef:        gen.Reference().Bytes(),
			AccountState:     gen.ID().Bytes(),
			Status:           "TEST",
			AccountRef:       gen.Reference().Bytes(),
		}
		err := db.Insert(memberModel)
		require.NoError(t, err)
	}

	t.Run("genesis, no lockup date, no vesting", func(t *testing.T) {
		for i := 0; i < size; i++ {
			depModel := &models.Deposit{
				EtheriumHash:    "genesis_deposit",
				MemberReference: gen.Reference().Bytes(),
				Reference:       gen.Reference().Bytes(),
				TransferDate:    now.Unix() - 1000,
				HoldReleaseDate: now.Unix() - 1000,
				Amount:          "10000",
				Balance:         "10000",
				State:           gen.ID().Bytes(),
				Vesting:         10, // seconds
				VestingStep:     10, // seconds
				InnerStatus:     models.DepositStatusConfirmed,
			}
			err := db.Insert(depModel)
			require.NoError(t, err)
		}

		stats := getStats(nil)
		require.Equal(t, "0", stats.Circulating())
		require.Equal(t, "100000", stats.Total())
	})

	t.Run("genesis, lockup date, vesting", func(t *testing.T) {
		//
		for i := 0; i < size; i++ {
			depModel := &models.Deposit{
				EtheriumHash:    "genesis_deposit",
				MemberReference: gen.Reference().Bytes(),
				Reference:       gen.Reference().Bytes(),
				TransferDate:    now.Unix(),
				HoldReleaseDate: now.Unix() + 365*24*3600,
				Amount:          "10000",
				Balance:         "10000",
				State:           gen.ID().Bytes(),
				Vesting:         1000000,
				VestingStep:     100,
				InnerStatus:     models.DepositStatusConfirmed,
			}
			err := db.Insert(depModel)
			require.NoError(t, err)
		}
		stats := getStats(nil)
		require.Equal(t, "0", stats.Circulating())
		require.Equal(t, "100000", stats.Total())
		require.Equal(t, "200000", stats.Max())

		// after lockup date, after partial vesting
		after := now.Add(365*24*time.Hour + 400000*time.Second)
		stats = getStats(&after)
		require.Equal(t, "0", stats.Circulating())
		require.Equal(t, "140000", stats.Total())
		require.Equal(t, "200000", stats.Max())
	})

	t.Run("user transfer deposit->wallet", func(t *testing.T) {
		for i := 0; i < size; i++ {
			memberModel := &postgres.MemberSchema{
				MemberRef:        gen.Reference().Bytes(),
				Balance:          "500",
				MigrationAddress: random.String(10),
				WalletRef:        gen.Reference().Bytes(),
				AccountState:     gen.ID().Bytes(),
				Status:           "TEST",
				AccountRef:       gen.Reference().Bytes(),
			}
			err := db.Insert(memberModel)
			require.NoError(t, err)
		}
		stats := getStats(nil)
		require.Equal(t, "5000", stats.Circulating())
		require.Equal(t, "105000", stats.Total())
		require.Equal(t, "200000", stats.Max())
	})

	t.Run("user migration deposits", func(t *testing.T) {
		for i := 0; i < size; i++ {
			depModel := &models.Deposit{
				EtheriumHash:    random.String(10),
				MemberReference: gen.Reference().Bytes(),
				Reference:       gen.Reference().Bytes(),
				TransferDate:    now.Unix(),
				HoldReleaseDate: now.Unix() + 30*24*3600,
				Amount:          "5000",
				Balance:         "5000",
				State:           gen.ID().Bytes(),
				Vesting:         30 * 24 * 3600,
				VestingStep:     3600,
				InnerStatus:     models.DepositStatusConfirmed,
			}
			err := db.Insert(depModel)
			require.NoError(t, err)
		}
		stats := getStats(nil)
		require.Equal(t, "5000", stats.Circulating())
		require.Equal(t, "105000", stats.Total())
		require.Equal(t, "200000", stats.Max())

		// after lockup date, after vesting
		after := now.Add(61 * 24 * time.Hour)
		stats = getStats(&after)
		require.Equal(t, "5000", stats.Circulating())
		require.Equal(t, "155000", stats.Total())
		require.Equal(t, "200000", stats.Max())
	})

	t.Run("partial user vesting cases", func(t *testing.T) {
		for i := 0; i < size; i++ {
			depModel := &models.Deposit{
				EtheriumHash:    random.String(10),
				MemberReference: gen.Reference().Bytes(),
				Reference:       gen.Reference().Bytes(),
				TransferDate:    now.Unix(),
				HoldReleaseDate: now.Unix() + 10,
				Amount:          "5000",
				Balance:         "5000",
				State:           gen.ID().Bytes(),
				Vesting:         1000,
				VestingStep:     50,
				InnerStatus:     models.DepositStatusConfirmed,
			}
			err := db.Insert(depModel)
			require.NoError(t, err)
		}
		stats := getStats(nil)
		require.Equal(t, "5000", stats.Circulating())
		require.Equal(t, "105000", stats.Total())
		require.Equal(t, "200000", stats.Max())

		// after lockup date, after half vesting
		after := now.Add(10*time.Second + 1000/2*time.Second)
		stats = getStats(&after)
		require.Equal(t, "5000", stats.Circulating())
		require.Equal(t, "130000", stats.Total())
		require.Equal(t, "200000", stats.Max())

		// after lockup date, after 0.655 of vesting
		after = now.Add(10*time.Second + 655*time.Second)
		stats = getStats(&after)
		require.Equal(t, "5000", stats.Circulating())
		require.Equal(t, "137500", stats.Total())
		require.Equal(t, "200000", stats.Max())

		// after lockup date, after x2 of vesting
		after = now.Add(10*time.Second + 1000*2*time.Second)
		stats = getStats(&after)
		require.Equal(t, "5000", stats.Circulating())
		require.Equal(t, "155000", stats.Total())
		require.Equal(t, "200000", stats.Max())

		// after lockup date, exact
		after = now.Add(10 * time.Second)
		stats = getStats(&after)
		require.Equal(t, "5000", stats.Circulating())
		require.Equal(t, "105000", stats.Total())
		require.Equal(t, "200000", stats.Max())

		// after lockup date, after vesting exact
		after = now.Add(10*time.Second + 1000*time.Second)
		stats = getStats(&after)
		require.Equal(t, "5000", stats.Circulating())
		require.Equal(t, "155000", stats.Total())
		require.Equal(t, "200000", stats.Max())
	})

	t.Run("partial user vesting cases, bad vesting step", func(t *testing.T) {
		_, err := db.Exec("truncate table members")
		require.NoError(t, err)
		_, err = db.Exec("truncate table deposits")
		require.NoError(t, err)
		for i := 0; i < size; i++ {
			depModel := &models.Deposit{
				EtheriumHash:    "genesis_deposit",
				MemberReference: gen.Reference().Bytes(),
				Reference:       gen.Reference().Bytes(),
				TransferDate:    now.Unix(),
				HoldReleaseDate: now.Unix(),
				Amount:          "10000",
				Balance:         "10000",
				State:           gen.ID().Bytes(),
				Vesting:         1000,
				VestingStep:     13,
				InnerStatus:     models.DepositStatusConfirmed,
			}
			err := db.Insert(depModel)
			require.NoError(t, err)
		}
		stats := getStats(nil)
		require.Equal(t, "0", stats.Circulating())
		require.Equal(t, "0", stats.Total())
		require.Equal(t, "100000", stats.Max())

		// after lockup date, after half vesting
		after := now.Add(500 * time.Second)
		stats = getStats(&after)
		require.Equal(t, "0", stats.Circulating())
		require.Equal(t, "49350", stats.Total())
		require.Equal(t, "100000", stats.Max())

		// after lockup date, after 0.655 of vesting
		after = now.Add(655 * time.Second)
		stats = getStats(&after)
		require.Equal(t, "0", stats.Circulating())
		require.Equal(t, "64930", stats.Total())
		require.Equal(t, "100000", stats.Max())

		// after lockup date, after x2 of vesting
		after = now.Add(1000 * 2 * time.Second)
		stats = getStats(&after)
		require.Equal(t, "0", stats.Circulating())
		require.Equal(t, "100000", stats.Total())
		require.Equal(t, "100000", stats.Max())

		// after lockup date, exact
		stats = getStats(&now)
		require.Equal(t, "0", stats.Circulating())
		require.Equal(t, "0", stats.Total())
		require.Equal(t, "100000", stats.Max())

		// after lockup date, after vesting exact
		after = now.Add(1000 * time.Second)
		stats = getStats(&after)
		require.Equal(t, "0", stats.Circulating())
		require.Equal(t, "100000", stats.Total())
		require.Equal(t, "100000", stats.Max())
	})
}
