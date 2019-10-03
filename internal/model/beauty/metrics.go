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

package beauty

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	membersDumped = stats.Int64(
		"dumped_members_count",
		"count of dumped member records",
		stats.UnitDimensionless,
	)

	deposits = stats.Int64(
		"dumped_deposits_count",
		"count of dumped deposit records",
		stats.UnitDimensionless,
	)

	MigrationAddresses = stats.Int64( // "Exported" for a hack. Cause - used in other component.
		"dumped_migration_addresses_count",
		"count of dumped migration addresses",
		stats.UnitDimensionless,
	)

	membersTransfers = stats.Int64(
		"dumped_members_transfers_count",
		"count of dumped member's transfers records",
		stats.UnitDimensionless,
	)

	pulsesDumped = stats.Int64(
		"dumped_pulses_count",
		"count of dumped pulses records",
		stats.UnitDimensionless,
	)

	wastedMigrationAddresses = stats.Int64(
		"wasted_migration_addresses_count",
		"count of wasted migration addresses",
		stats.UnitDimensionless,
	)

	depositsUpdated = stats.Int64(
		"updated_deposits_count",
		"count of updated deposits",
		stats.UnitDimensionless,
	)

	balancesUpdated = stats.Int64(
		"updated_balances_count",
		"count of updated balances",
		stats.UnitDimensionless,
	)
)

func init() {
	err := view.Register(
		&view.View{
			Name:        membersDumped.Name(),
			Description: membersDumped.Description(),
			Measure:     membersDumped,
			Aggregation: view.Count(),
		},

		&view.View{
			Name:        deposits.Name(),
			Description: deposits.Description(),
			Measure:     deposits,
			Aggregation: view.Count(),
		},

		&view.View{
			Name:        MigrationAddresses.Name(),
			Description: MigrationAddresses.Description(),
			Measure:     MigrationAddresses,
			Aggregation: view.Count(),
		},

		&view.View{
			Name:        membersTransfers.Name(),
			Description: membersTransfers.Description(),
			Measure:     membersTransfers,
			Aggregation: view.Count(),
		},

		&view.View{
			Name:        pulsesDumped.Name(),
			Description: pulsesDumped.Description(),
			Measure:     pulsesDumped,
			Aggregation: view.Count(),
		},

		&view.View{
			Name:        wastedMigrationAddresses.Name(),
			Description: wastedMigrationAddresses.Description(),
			Measure:     wastedMigrationAddresses,
			Aggregation: view.Count(),
		},

		&view.View{
			Name:        depositsUpdated.Name(),
			Description: depositsUpdated.Description(),
			Measure:     depositsUpdated,
			Aggregation: view.Count(),
		},

		&view.View{
			Name:        balancesUpdated.Name(),
			Description: balancesUpdated.Description(),
			Measure:     balancesUpdated,
			Aggregation: view.Count(),
		},
	)
	if err != nil {
		panic(err)
	}
}
