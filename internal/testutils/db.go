package testutils

import (
	"fmt"
	"log"

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
