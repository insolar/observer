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

package migration

import (
	"github.com/insolar/insolar/instrumentation/insmetrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	migration = insmetrics.MustTagKey("migration_cache")
)

var (
	migrationAddressCache = stats.Int64(
		"migration_address_cache",
		"count of migration address's cached utility records",
		stats.UnitDimensionless,
	)
	migrationKeeperCache = stats.Int64(
		"migration_keeper_cache",
		"count of migration address keeper's cached utility records",
		stats.UnitDimensionless,
	)
	migrationAddresses = stats.Int64(
		"migration_addresses_total",
		"count of migration addresses in db",
		stats.UnitDimensionless,
	)
	migrationAddressDefers = stats.Int64(
		"migration_address_defers",
		"count of deferred migration addresses",
		stats.UnitDimensionless,
	)
)

func init() {
	err := view.Register(
		&view.View{
			Name:        migrationAddressCache.Name(),
			Description: migrationAddressCache.Description(),
			Measure:     migrationAddressCache,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{migration},
		},
		&view.View{
			Name:        migrationKeeperCache.Name(),
			Description: migrationKeeperCache.Description(),
			Measure:     migrationKeeperCache,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{migration},
		},
		&view.View{
			Name:        migrationAddresses.Name(),
			Description: migrationAddresses.Description(),
			Measure:     migrationAddresses,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{migration},
		},
		&view.View{
			Name:        migrationAddressDefers.Name(),
			Description: migrationAddressDefers.Description(),
			Measure:     migrationAddressDefers,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{migration},
		},
	)
	if err != nil {
		panic(err)
	}
}
