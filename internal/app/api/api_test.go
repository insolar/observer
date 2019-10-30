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

package api

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/api/internalapi"
	"github.com/insolar/observer/internal/app/api/observerapi"
)

const (
	database      = "test_api_db"
	migrationsDir = "../../../scripts/migrations"
	password      = "secret"

	apihost = ":14800"
)

var (
	db *pg.DB

	pgOptions = &pg.Options{
		Addr:            "localhost",
		User:            "postgres",
		Password:        password,
		Database:        database,
		ApplicationName: "observer",
	}
)

func TestMain(t *testing.M) {
	var err error
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.Run("postgres", "11", []string{"POSTGRES_PASSWORD=" + password, "POSTGRES_DB=" + database})
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

	err = migrationCollection.DiscoverSQLMigrations(migrationsDir)
	if err != nil {
		log.Panicf("Failed to read migrations: %s", err)
	}

	_, _, err = migrationCollection.Run(db, "up")
	if err != nil {
		log.Panicf("Could not migrate: %s", err)
	}

	e := echo.New()

	logger := logrus.New()
	observerAPI := NewObserverServer(db, logger)
	observerapi.RegisterHandlers(e, observerAPI)
	internalapi.RegisterHandlers(e, observerAPI)
	go func() {
		e.Logger.Fatal(e.Start(apihost))
	}()
	// TODO: wait until API started
	// TODO: flush db
	time.Sleep(5 * time.Second)
	os.Exit(t.Run())
}
