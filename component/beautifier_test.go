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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/insolar/insolar/application/api/requester"
	"github.com/insolar/insolar/application/builtin/contract/deposit"
	"github.com/insolar/insolar/application/builtin/proxy/member"
	"github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/collecting"
	"github.com/insolar/observer/observability"
)

var (
	db       *pg.DB
	database = "test_beautifier_db"

	pgOptions = &pg.Options{
		Addr:            "localhost",
		User:            "postgres",
		Password:        "secret",
		Database:        "test_beautifier_db",
		ApplicationName: "observer",
	}
)

func TestMain(t *testing.M) {
	var err error
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.Run("postgres", "11", []string{"POSTGRES_PASSWORD=secret", "POSTGRES_DB=" + database})
	if err != nil {
		log.Panicf("Could not start resource: %s", err)
	}

	defer func() {
		// When you're done, kill and remove the container
		err = pool.Purge(resource)
		if err != nil {
			log.Panicf("failed to purge docker pool: %s", err)
		}
	}()

	if err = pool.Retry(func() error {
		options := *pgOptions
		options.Addr = fmt.Sprintf("%s:%s", options.Addr, resource.GetPort("5432/tcp"))
		db = pg.Connect(&options)
		_, err := db.Exec("select 1")
		return err
	}); err != nil {
		log.Panicf("Could not connect to docker: %s", err)
	}
	defer db.Close()

	migrationCollection := migrations.NewCollection()

	_, _, err = migrationCollection.Run(db, "init")
	if err != nil {
		log.Panicf("Could not init migrations: %s", err)
	}

	err = migrationCollection.DiscoverSQLMigrations("../scripts/migrations")
	if err != nil {
		log.Panicf("Failed to read migrations: %s", err)
	}

	_, _, err = migrationCollection.Run(db, "up")
	if err != nil {
		log.Panicf("Could not migrate: %s", err)
	}

	os.Exit(t.Run())
}

type fakeConn struct {
}

func (f fakeConn) PG() *pg.DB {
	return db
}

func TestBeautifier_Run(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		cfg := &configuration.Configuration{
			Replicator: configuration.Replicator{
				CacheSize: 100000,
			},
			LogLevel: "debug",
		}
		beautifier := makeBeautifier(cfg, observability.Make(cfg), fakeConn{})
		ctx := context.Background()

		beautifier(ctx, nil)
	})

	t.Run("happy path", func(t *testing.T) {
		cfg := &configuration.Configuration{
			Replicator: configuration.Replicator{
				CacheSize: 100000,
			},
			LogLevel: "debug",
		}
		beautifier := makeBeautifier(cfg, observability.Make(cfg), fakeConn{})
		ctx := context.Background()

		tdg := NewTreeDataGenerator()
		raw := &raw{
			batch: []*observer.Record{
				tdg.makeRequestWith("hello", gen.RecordReference(), nil),
			},
		}
		res := beautifier(ctx, raw)
		assert.NotNil(t, res)
	})

	t.Run("wastings", func(t *testing.T) {
		cfg := &configuration.Configuration{
			Replicator: configuration.Replicator{
				CacheSize: 100000,
			},
			LogLevel: "debug",
		}
		beautifier := makeBeautifier(cfg, observability.Make(cfg), fakeConn{})
		ctx := context.Background()

		tdg := NewTreeDataGenerator()

		pn := insolar.GenesisPulse.PulseNumber
		address := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"
		out := tdg.makeOutgouingRequest(gen.Reference(), gen.Reference())
		call := tdg.makeGetMigrationAddressCall(pn)

		raw := &raw{
			batch: []*observer.Record{
				out,
				call,
				tdg.makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
				tdg.makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{address, nil}}),
			},
		}
		res := beautifier(ctx, raw)
		assert.Equal(t, map[string]*observer.Wasting{
			address: {
				Addr: address,
			}}, res.wastings)
	})

	t.Run("transfer happy path", func(t *testing.T) {
		cfg := &configuration.Configuration{
			Replicator: configuration.Replicator{
				CacheSize: 100000,
			},
			LogLevel: "debug",
		}
		beautifier := makeBeautifier(cfg, observability.Make(cfg), fakeConn{})
		ctx := context.Background()
		tdg := NewTreeDataGenerator()

		pn := insolar.GenesisPulse.PulseNumber
		amount := "42"
		fee := "7"
		from := gen.IDWithPulse(pn)
		to := gen.IDWithPulse(pn)
		call := makeTransferCall(amount, from.String(), to.String(), pn)
		out := tdg.makeOutgouingRequest(gen.Reference(), gen.Reference())
		timestamp, err := pn.AsApproximateTime()
		if err != nil {
			panic("failed to calc timestamp by pulse")
		}

		expected := []*observer.ExtendedTransfer{
			{
				DepositTransfer: observer.DepositTransfer{
					Transfer: observer.Transfer{
						TxID:      call.ID,
						From:      from,
						To:        to,
						Amount:    amount,
						Fee:       fee,
						Pulse:     pn,
						Timestamp: timestamp.Unix(),
					},
				},
			},
			{
				DepositTransfer: observer.DepositTransfer{
					Transfer: observer.Transfer{
						TxID:      call.ID,
						Timestamp: timestamp.Unix(),
						Pulse:     call.ID.Pulse(),
						Status:    "FAILED",
					},
					EthHash: "",
				},
			},
		}

		raw := &raw{
			batch: []*observer.Record{
				out,
				call,
				tdg.makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
				tdg.makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{&member.TransferResponse{Fee: fee}, nil}}),
			},
		}
		res := beautifier(ctx, raw)
		assert.Equal(t, expected, res.transfers)
	})

	t.Run("deposit", func(t *testing.T) {
		cfg := &configuration.Configuration{
			Replicator: configuration.Replicator{
				CacheSize: 100000,
			},
			LogLevel: "debug",
		}
		beautifier := makeBeautifier(cfg, observability.Make(cfg), fakeConn{})
		ctx := context.Background()
		pn := insolar.GenesisPulse.PulseNumber
		tdg := NewTreeDataGenerator()

		call := tdg.makeDepositMigrationCall(pn)

		memberRef := gen.Reference()
		out := tdg.makeOutgouingRequest(*insolar.NewReference(call.ID), gen.Reference())
		in := tdg.makeIncomingFromOutgoing(out.Virtual.Union.(*record.Virtual_OutgoingRequest).OutgoingRequest)

		balance := "123"
		amount := "456"
		txHash := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"

		dep := deposit.Deposit{
			Balance:            balance,
			Amount:             amount,
			TxHash:             txHash,
			PulseDepositUnHold: pn + 3,
		}
		memory, err := insolar.Serialize(dep)
		if err != nil {
			panic("fail serialize memory")
		}

		act := tdg.makeActivation(
			*insolar.NewReference(in.ID),
			*migrationdaemon.PrototypeReference,
			memory,
		)

		raw := &raw{
			batch: []*observer.Record{
				call,
				out,
				tdg.makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
				in,
				tdg.makeResultWith(in.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
				act,
				tdg.makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{
					migrationdaemon.DepositMigrationResult{Reference: memberRef.String()},
					nil,
				}}),
			},
		}
		transferDate, err := act.ID.Pulse().AsApproximateTime()
		require.NoError(t, err)

		res := beautifier(ctx, raw)

		assert.Equal(t, 1, len(res.deposits))
		assert.Equal(t, map[insolar.ID]*observer.Deposit{
			act.ID: {
				EthHash:         strings.ToLower(txHash),
				Ref:             in.ID,                 // from activate
				Member:          *memberRef.GetLocal(), // from result
				Timestamp:       transferDate.Unix(),
				HoldReleaseDate: 0,
				Amount:          amount,  // from activate
				Balance:         balance, // from activate
				DepositState:    act.ID,  // from activate
			},
		}, res.deposits)
	})
}

type treeDataGenerator struct {
	Nonce uint64
}

func NewTreeDataGenerator() treeDataGenerator {
	return treeDataGenerator{Nonce: 0}
}

// not thread safe
func (t *treeDataGenerator) GetNonce() uint64 {
	nonce := t.Nonce
	t.Nonce++
	return nonce
}

func (t *treeDataGenerator) makeRequestWith(method string, reason insolar.Reference, args []byte) *observer.Record {
	return &observer.Record{
		ID: gen.ID(),
		Virtual: record.Virtual{Union: &record.Virtual_IncomingRequest{IncomingRequest: &record.IncomingRequest{
			Method:    method,
			Reason:    reason,
			Arguments: args,
			Nonce:     t.GetNonce(),
		}}}}
}

// we need reasn for match too tree and some UNIQUE nonce
func (t *treeDataGenerator) makeOutgouingRequest(reason insolar.Reference, prototypeRef insolar.Reference) *observer.Record {
	rec := &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_OutgoingRequest{
				OutgoingRequest: &record.OutgoingRequest{
					Reason:    reason,
					Prototype: &prototypeRef,
					Nonce:     t.GetNonce(),
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

// we need same nonce in makeOutgouingRequest and makeIncomingRequest
func (t *treeDataGenerator) makeIncomingFromOutgoing(outgoing *record.OutgoingRequest) *observer.Record {
	rec := &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Reason:    outgoing.Reason,
					Nonce:     outgoing.Nonce,
					Prototype: outgoing.Prototype,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func (t *treeDataGenerator) makeActivation(ref insolar.Reference, prototypreRef insolar.Reference, memory []byte) *observer.Record {
	rec := &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &record.Activate{
					Request: ref,
					Image:   prototypreRef,
					Memory:  memory,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func (t *treeDataGenerator) makeGetMigrationAddressCall(pn insolar.PulseNumber) *observer.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}

	virtRecord := record.Wrap(&record.IncomingRequest{
		Method:    collecting.GetFreeMigrationAddress,
		Arguments: args,
	})

	rec := &record.Material{
		ID:      gen.IDWithPulse(pn),
		Virtual: virtRecord,
	}
	return (*observer.Record)(rec)
}

func (t *treeDataGenerator) makeDepositMigrationCall(pn insolar.PulseNumber) *observer.Record {
	request := &requester.ContractRequest{
		Params: requester.Params{
			CallSite:   collecting.CallSite,
			CallParams: nil,
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
					Nonce:     t.GetNonce(),
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func (t *treeDataGenerator) makeResultWith(requestID insolar.ID, result *foundation.Result) *observer.Record {
	payload, err := insolar.Serialize(result)
	if err != nil {
		panic("failed to serialize result")
	}
	ref := insolar.NewReference(requestID)
	rec := &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_Result{
				Result: &record.Result{
					Request: *ref,
					Payload: payload,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeTransferCall(amount, from, to string, pulse insolar.PulseNumber) *observer.Record {
	request := &requester.ContractRequest{
		Params: requester.Params{
			CallSite: collecting.TransferMethod,
			CallParams: collecting.TransferCallParams{
				Amount:            amount,
				ToMemberReference: to,
			},
			Reference: from,
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
		ID: gen.IDWithPulse(pulse),
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
