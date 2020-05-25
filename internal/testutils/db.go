// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package testutils

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/ory/dockertest/v3"
)

var pgOptions = &pg.Options{
	Addr:            "localhost",
	Database:        "observer_test_db",
	User:            "postgres",
	Password:        "secret",
	ApplicationName: "observer",
}

func SetupDB(migrationsDir string) (*pg.DB, pg.Options, func()) {
	var err error
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.Run(
		"postgres", "11",
		[]string{
			"POSTGRES_DB=" + pgOptions.Database,
			"POSTGRES_PASSWORD=" + pgOptions.Password,
		},
	)
	if err != nil {
		log.Panicf("Could not start resource: %s", err)
	}

	poolCleaner := func() {
		// When you're done, kill and remove the container
		log.Printf("removing container")
		err := pool.Purge(resource)
		if err != nil {
			log.Printf("failed to purge docker pool: %s", err)
		}
	}

	options := *pgOptions
	options.Addr = fmt.Sprintf("%s:%s", options.Addr, resource.GetPort("5432/tcp"))

	var db *pg.DB
	err = pool.Retry(func() error {
		db = pg.Connect(&options)
		_, err := db.Exec("select 1")
		return err
	})
	if err != nil {
		poolCleaner()
		log.Panicf("Could not start postgres: %s", err)
	}

	dbCleaner := func() {
		log.Printf("shutting down db")
		err := db.Close()
		if err != nil {
			log.Printf("failed to purge docker pool: %s", err)
		}
	}
	cleaner := func() {
		dbCleaner()
		poolCleaner()
	}

	migrationCollection := migrations.NewCollection()

	_, _, err = migrationCollection.Run(db, "init")
	if err != nil {
		cleaner()
		log.Panicf("Could not init migrations: %s", err)
	}

	err = migrationCollection.DiscoverSQLMigrations(migrationsDir)
	if err != nil {
		cleaner()
		log.Panicf("Failed to read migrations: %s", err)
	}

	_, _, err = migrationCollection.Run(db, "up")
	if err != nil {
		cleaner()
		log.Panicf("Could not migrate: %s", err)
	}
	return db, options, cleaner
}

func TruncateTables(t *testing.T, db *pg.DB, models []interface{}) {
	for _, m := range models {
		_, err := db.Model(m).Exec("TRUNCATE TABLE ?TableName CASCADE")
		require.NoError(t, err)
	}
}
