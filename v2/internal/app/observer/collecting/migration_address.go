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

package collecting

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/migrationshard"
	proxyShard "github.com/insolar/insolar/logicrunner/builtin/proxy/migrationshard"

	"github.com/insolar/observer/v2/internal/app/observer"
	"github.com/insolar/observer/v2/internal/pkg/panic"
)

type MigrationAddressCollector struct {
	results observer.ResultCollector
}

func NewMigrationAddressesCollector() *MigrationAddressCollector {
	results := NewResultCollector(isAddMigrationAddresses, successResult)
	return &MigrationAddressCollector{
		results: results,
	}
}

func (c *MigrationAddressCollector) Collect(rec *observer.Record) []*observer.MigrationAddress {
	defer panic.Catch("migration_address_collector")

	if rec == nil {
		return nil
	}

	// This code block collects addresses from incoming request.
	couple := c.results.Collect(rec)
	if couple != nil {
		result := couple.Result
		if !result.IsSuccess() {
			return nil
		}
		request := couple.Request
		params := &addAddresses{}
		request.ParseMemberContractCallParams(params)
		addresses := []*observer.MigrationAddress{}
		for _, addr := range params.MigrationAddresses {
			addresses = append(addresses, &observer.MigrationAddress{
				Addr:  addr,
				Pulse: request.ID.Pulse(),
			})
		}
		return addresses
	}

	// This code block collects addresses from genesis record.
	activate := observer.CastToActivate(rec)
	if !activate.IsActivate() {
		return nil
	}

	act := activate.Virtual.GetActivate()
	if !isMigrationShardActivate(act) {
		return nil
	}
	shard := migrationShardActivate(act)
	addresses := []*observer.MigrationAddress{}
	for _, addr := range shard {
		addresses = append(addresses, &observer.MigrationAddress{
			Addr:  addr,
			Pulse: rec.ID.Pulse(),
		})
	}
	return addresses
}

type addAddresses struct {
	MigrationAddresses []string `json:"migrationAddresses"`
}

func isMigrationShardActivate(act *record.Activate) bool {
	return act.Image.Equal(*proxyShard.PrototypeReference)
}

func migrationShardActivate(act *record.Activate) []string {
	if act.Memory == nil {
		return []string{}
	}
	shard := &migrationshard.MigrationShard{}
	err := insolar.Deserialize(act.Memory, shard)
	if err != nil {
		return []string{}
	}
	return shard.FreeMigrationAddresses
}

func isAddMigrationAddresses(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	if !request.IsMemberCall() {
		return false
	}

	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "migration.addAddresses"
}
