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

	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
)

func TestSupplyStats(t *testing.T) {
	repo := postgres.NewSupplyStatsRepository(db)

	stats, err := repo.CountStats()
	require.NoError(t, err)
	require.Equal(t, "0", stats.Total)

	member := models.Member{
		Reference: gen.Reference().Bytes(),
		Balance:   "1234567890",
	}
	err = db.Insert(&member)
	require.NoError(t, err)

	stats, err = repo.CountStats()
	require.NoError(t, err)
	require.Equal(t, "1234567890", stats.Total)

	stats, err = repo.LastStats()
	require.Error(t, err)

	stats, err = repo.CountStats()
	require.NoError(t, err)
	err = repo.InsertStats(stats)
	require.NoError(t, err)

	stats, err = repo.LastStats()
	require.NoError(t, err)
	require.Equal(t, "1234567890", stats.Total)
}
