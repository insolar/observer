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
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/models"
	"github.com/insolar/observer/internal/testutils"

	"github.com/go-pg/pg"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

const (
	migrationsDir = "../../../scripts/migrations"

	apihost = ":14800"
)

var (
	db *pg.DB

	clock = &testClock{}

	testFee = big.NewInt(1000000000)
	testPrice = "0.05"
)

func TestMain(t *testing.M) {

	var dbCleaner func()
	db, _, dbCleaner = testutils.SetupDB(migrationsDir)

	e := echo.New()

	logger := logrus.New()
	observerAPI := NewObserverServer(db, logger, testFee, clock, testPrice)
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
}

type testClock struct {
	nowTime int64
}

func (c *testClock) Now() time.Time {
	return time.Unix(c.nowTime, 0)
}
