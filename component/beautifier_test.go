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
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"

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

		raw := &raw{
			batch: []*observer.Record{
				makeRequestWith("hello", gen.RecordReference(), nil),
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

		pn := insolar.GenesisPulse.PulseNumber
		address := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"
		out := makeOutgouingRequest()
		call := makeGetMigrationAddressCall(pn)

		raw := &raw{
			batch: []*observer.Record{
				out,
				call,
				makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
				makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{address, nil}}),
			},
		}
		res := beautifier(ctx, raw)
		assert.Equal(t, map[string]*observer.Wasting{
			address: {
				Addr: address,
			}}, res.wastings)
	})
}

func makeRequestWith(method string, reason insolar.Reference, args []byte) *observer.Record {
	return &observer.Record{
		ID: gen.ID(),
		Virtual: record.Virtual{Union: &record.Virtual_IncomingRequest{IncomingRequest: &record.IncomingRequest{
			Method:    method,
			Reason:    reason,
			Arguments: args,
		}}}}
}

func makeOutgouingRequest() *observer.Record {
	rec := &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_OutgoingRequest{
				OutgoingRequest: &record.OutgoingRequest{},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeGetMigrationAddressCall(pn insolar.PulseNumber) *observer.Record {
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

func makeResultWith(requestID insolar.ID, result *foundation.Result) *observer.Record {
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
