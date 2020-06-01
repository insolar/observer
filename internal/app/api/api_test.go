// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package api

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/pulse"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/insolar/insolar/instrumentation/inslogger"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
	"github.com/insolar/observer/internal/testutils"
)

const (
	migrationsDir = "../../../scripts/migrations"

	apihost = ":14800"
)

var (
	db *pg.DB

	pStorage observer.PulseStorage

	testFee   = big.NewInt(1000000000)
	testPrice = "0.05"
)

func TestMain(t *testing.M) {

	var dbCleaner func()
	db, _, dbCleaner = testutils.SetupDB(migrationsDir)

	e := echo.New()

	logger := inslogger.FromContext(context.Background())

	pStorage = postgres.NewPulseStorage(logger, db)
	nowPulse := 1575302444 - pulse.UnixTimeOfMinTimePulse + pulse.MinTimePulse
	_ = pStorage.Insert(&observer.Pulse{Number: pulse.Number(nowPulse)})

	observerAPI := NewServer(db, logger, pStorage, getApiConfig())

	RegisterHandlers(e, observerAPI)
	go func() {
		err := e.Start(apihost)
		dbCleaner()
		e.Logger.Fatal(err)
	}()
	// TODO: wait until API started
	time.Sleep(5 * time.Second)

	retCode := t.Run()

	dbCleaner()
	os.Exit(retCode)
}

func truncateDB(t *testing.T) {
	_, err := db.Model(&models.Transaction{}).Exec("TRUNCATE TABLE ?TableName CASCADE")
	require.NoError(t, err)
	_, err = db.Model(&models.Member{}).Exec("TRUNCATE TABLE ?TableName CASCADE")
	require.NoError(t, err)
	_, err = db.Model(&models.Deposit{}).Exec("TRUNCATE TABLE ?TableName CASCADE")
	require.NoError(t, err)
	_, err = db.Model(&models.MigrationAddress{}).Exec("TRUNCATE TABLE ?TableName CASCADE")
	require.NoError(t, err)

	_, err = db.Exec("TRUNCATE TABLE pulses CASCADE")
	require.NoError(t, err)
	nowPulse := 1575302444 - pulse.UnixTimeOfMinTimePulse + pulse.MinTimePulse
	_ = pStorage.Insert(&observer.Pulse{Number: pulse.Number(nowPulse)})
}
