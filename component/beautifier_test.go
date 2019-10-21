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
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/require"
)

func Test_SortByType(t *testing.T) {
	var batch []*observer.Record
	batch = append(batch,
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Deactivate{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Result{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_OutgoingRequest{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_IncomingRequest{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Activate{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Code{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Amend{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_PendingFilament{}}},
	)

	var expected []*observer.Record
	expected = append(expected,
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Code{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_PendingFilament{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_IncomingRequest{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_OutgoingRequest{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Activate{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Amend{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Deactivate{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Result{}}},
	)

	// not random but shuffled
	sort.Slice(batch, func(i, j int) bool {
		return TypeOrder(batch[i]) < TypeOrder(batch[j])
	})
	require.Equal(t, expected, batch)

	// real random
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(batch), func(i, j int) { batch[i], batch[j] = batch[j], batch[i] })
	sort.Slice(batch, func(i, j int) bool {
		return TypeOrder(batch[i]) < TypeOrder(batch[j])
	})
	require.Equal(t, expected, batch)
}

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
