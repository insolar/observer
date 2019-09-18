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
	"encoding/json"
	"testing"

	"github.com/insolar/insolar/api/requester"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/migrationshard"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/stretchr/testify/require"

	proxyShard "github.com/insolar/insolar/logicrunner/builtin/proxy/migrationshard"

	"github.com/insolar/observer/v2/internal/app/observer"
)

func makeEmptyMigrationShard() *observer.Record {
	pn := insolar.GenesisPulse.PulseNumber
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &record.Activate{
					Request: gen.ReferenceWithPulse(pn),
					Memory:  nil,
					Image:   *proxyShard.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeInvalidMigrationShard() *observer.Record {
	pn := insolar.GenesisPulse.PulseNumber
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &record.Activate{
					Request: gen.ReferenceWithPulse(pn),
					Memory:  []byte{1, 2, 3},
					Image:   *proxyShard.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeGenesisMigrationAddresses() ([]*observer.MigrationAddress, *observer.Record) {
	addrs := []string{
		"test_address",
	}
	pn := insolar.GenesisPulse.PulseNumber
	addresses := []*observer.MigrationAddress{}
	for _, a := range addrs {
		addresses = append(addresses, &observer.MigrationAddress{a, pn})
	}

	shard := &migrationshard.MigrationShard{FreeMigrationAddresses: addrs}
	memory, err := insolar.Serialize(shard)
	if err != nil {
		panic("failed to serialize migration address shard memory")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &record.Activate{
					Request: gen.ReferenceWithPulse(pn),
					Memory:  memory,
					Image:   *proxyShard.PrototypeReference,
				},
			},
		},
	}
	return addresses, (*observer.Record)(rec)
}

func makeAddRequest(pn insolar.PulseNumber, addrs []string) *observer.Record {
	request := &requester.ContractRequest{
		Params: requester.Params{
			CallSite: "migration.addAddresses",
			CallParams: addAddresses{
				MigrationAddresses: addrs,
			},
		},
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		panic("failed to marshal request")
	}
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{requestBody, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "Call",
					Arguments: args,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeAddAddresses() ([]*observer.MigrationAddress, []*observer.Record) {
	addrs := []string{
		"test_address",
	}
	pn := insolar.GenesisPulse.PulseNumber
	addresses := []*observer.MigrationAddress{}
	for _, a := range addrs {
		addresses = append(addresses, &observer.MigrationAddress{a, pn})
	}

	out := makeOutgouingRequest()
	add := makeAddRequest(pn, addrs)

	records := []*observer.Record{
		out,
		makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		add,
		makeResultWith(add.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
	}
	return addresses, records
}

func TestMigrationAddressCollector_Collect(t *testing.T) {
	collector := NewMigrationAddressesCollector()

	t.Run("nil", func(t *testing.T) {
		require.Empty(t, collector.Collect(nil))
	})

	t.Run("add_addresses_request", func(t *testing.T) {
		expected, records := makeAddAddresses()
		actual := []*observer.MigrationAddress{}
		for _, rec := range records {
			addr := collector.Collect(rec)
			if addr != nil {
				actual = append(actual, addr...)
			}
		}

		require.Equal(t, expected, actual)
	})

	t.Run("not_addresses_activate", func(t *testing.T) {
		rec := makeAccountActivate(gen.PulseNumber(), "", gen.Reference())
		require.Empty(t, collector.Collect(rec))
	})

	t.Run("empty_migration_shard", func(t *testing.T) {
		rec := makeEmptyMigrationShard()
		require.Empty(t, collector.Collect(rec))
	})

	t.Run("invalid_migration_shard", func(t *testing.T) {
		rec := makeInvalidMigrationShard()
		require.Empty(t, collector.Collect(rec))
	})

	t.Run("genesis_address_pack", func(t *testing.T) {
		expected, rec := makeGenesisMigrationAddresses()

		require.Equal(t, expected, collector.Collect(rec))
	})
}
