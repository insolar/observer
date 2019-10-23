package api

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	database      = "test_api_db"
	migrationsDir = "../../../scripts/migrations"
	password      = "secret"
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

	os.Exit(t.Run())
}

func Test(t *testing.T) {
	expectedTransaction := transaction{
		PulseNumber: 1,
		Type:        TTypeMigration,
		Status:      TStatusSent,
		Amount:      "10",
		Fee:         "1",
	}
	_, err := db.Exec(
		`insert into simple_transactions (pulse_number, type, status, amount, fee) values (?, ?, ?, ?, ?)`,
		expectedTransaction.PulseNumber,
		expectedTransaction.Type,
		expectedTransaction.Status,
		expectedTransaction.Amount,
		expectedTransaction.Fee,
	)
	require.NoError(t, err)

	receivedTransaction := transaction{}
	_, err = db.QueryOne(&receivedTransaction, "select * from simple_transactions")
	require.NoError(t, err)

	receivedTransaction.ID = 0
	assert.Equal(t, expectedTransaction, receivedTransaction)
}
