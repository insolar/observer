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

package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	MembersCacheCount = stats.Int64(
		"members_cache_count",
		"count of member's cached utility records",
		stats.UnitDimensionless,
	)

	MigrationAddressCache = stats.Int64(
		"migration_address_cache",
		"count of migration address's cached utility records",
		stats.UnitDimensionless,
	)

	MigrationKeeperCache = stats.Int64(
		"migration_keeper_cache",
		"count of migration address keeper's cached utility records",
		stats.UnitDimensionless,
	)

	TransferCacheCount = stats.Int64(
		"transfer_cache_count",
		"count of transfer's cached utility records",
		stats.UnitDimensionless,
	)
)

func init() {
	err := view.Register(
		&view.View{
			Name:        MembersCacheCount.Name(),
			Description: MembersCacheCount.Description(),
			Measure:     MembersCacheCount,
			Aggregation: view.LastValue(),
		},

		&view.View{
			Name:        MigrationAddressCache.Name(),
			Description: MigrationAddressCache.Description(),
			Measure:     MigrationAddressCache,
			Aggregation: view.LastValue(),
		},

		&view.View{
			Name:        MigrationKeeperCache.Name(),
			Description: MigrationKeeperCache.Description(),
			Measure:     MigrationKeeperCache,
			Aggregation: view.LastValue(),
		},

		&view.View{
			Name:        TransferCacheCount.Name(),
			Description: TransferCacheCount.Description(),
			Measure:     TransferCacheCount,
			Aggregation: view.LastValue(),
		},
	)
	if err != nil {
		panic(err)
	}
}
