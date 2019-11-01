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
		require.Equal(t, "0.0000001000", res.Total)
		require.Equal(t, "0.0000002000", res.Max)
		require.Equal(t, "0.0000000100", res.Circulating)
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
		require.Equal(t, "11112.2220000000", res.Total)
		require.Equal(t, "33333.1111222222", res.Max)
		require.Equal(t, "44444.4111111111", res.Circulating)
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
		require.Equal(t, "11111222200000.0055555555", res.Total)
		require.Equal(t, "33333311112222.2244444444", res.Max)
		require.Equal(t, "44444441111111.1199999999", res.Circulating)
	})
}
