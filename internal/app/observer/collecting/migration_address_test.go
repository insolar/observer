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
	"context"
	"encoding/json"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/insolar/application/api/requester"
	"github.com/insolar/insolar/application/builtin/contract/migrationshard"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/stretchr/testify/require"

	proxyShard "github.com/insolar/insolar/application/builtin/proxy/migrationshard"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
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
		addresses = append(addresses, &observer.MigrationAddress{a, pn, false})
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

func makeAddRequest(pn insolar.PulseNumber, addrs []string) *record.Material {
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
	return &record.Material{
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
}

func TestMigrationAddressCollector_Collect(t *testing.T) {

	table := []struct {
		name  string
		mocks func(t minimock.Tester) (
			stream []*observer.Record, fetcher store.RecordFetcher, expectedResult []*observer.MigrationAddress,
		)
	}{
		{
			name: "nil",
			mocks: func(t minimock.Tester) ([]*observer.Record, store.RecordFetcher, []*observer.MigrationAddress) {
				fetcher := store.NewRecordFetcherMock(t)
				return []*observer.Record{nil}, fetcher, []*observer.MigrationAddress{}
			},
		},
		{
			name: "add_addresses_request",
			mocks: func(t minimock.Tester) ([]*observer.Record, store.RecordFetcher, []*observer.MigrationAddress) {
				fetcher := store.NewRecordFetcherMock(t)
				pn := insolar.GenesisPulse.PulseNumber

				addresses := []*observer.MigrationAddress{
					&observer.MigrationAddress{"test_address", pn, false},
				}

				add := makeAddRequest(pn, []string{"test_address"})
				fetcher.RequestMock.Return(*add, nil)
				records := []*observer.Record{
					(*observer.Record)(add),
					makeResultWith(add.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
				}
				return records, fetcher, addresses
			},
		},
		{
			name: "not_addresses_activate",
			mocks: func(t minimock.Tester) ([]*observer.Record, store.RecordFetcher, []*observer.MigrationAddress) {
				fetcher := store.NewRecordFetcherMock(t)
				rec, _ := makeAccountActivate(gen.PulseNumber(), "", gen.Reference())
				return []*observer.Record{rec}, fetcher, []*observer.MigrationAddress{}
			},
		},
		{
			name: "empty migration shard",
			mocks: func(t minimock.Tester) ([]*observer.Record, store.RecordFetcher, []*observer.MigrationAddress) {
				fetcher := store.NewRecordFetcherMock(t)
				rec := makeEmptyMigrationShard()
				return []*observer.Record{rec}, fetcher, []*observer.MigrationAddress{}
			},
		},
		{
			name: "invalid migration shard",
			mocks: func(t minimock.Tester) ([]*observer.Record, store.RecordFetcher, []*observer.MigrationAddress) {
				fetcher := store.NewRecordFetcherMock(t)
				rec := makeInvalidMigrationShard()
				return []*observer.Record{rec}, fetcher, []*observer.MigrationAddress{}
			},
		},
		{
			name: "genesis_address_pack",
			mocks: func(t minimock.Tester) ([]*observer.Record, store.RecordFetcher, []*observer.MigrationAddress) {
				fetcher := store.NewRecordFetcherMock(t)

				expected, rec := makeGenesisMigrationAddresses()
				return []*observer.Record{rec}, fetcher, expected
			},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			mc := minimock.NewController(t)
			records, fetcher, expected := test.mocks(mc)

			collector := NewMigrationAddressesCollector(inslogger.FromContext(ctx), fetcher)

			actual := make([]*observer.MigrationAddress, 0)
			for _, rec := range records {
				addr := collector.Collect(ctx, rec)
				if addr != nil {
					actual = append(actual, addr...)
				}
			}

			require.Equal(t, expected, actual)
			mc.Finish()
		})
	}
}
