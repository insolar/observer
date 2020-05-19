// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres_test

import (
	"os"
	"testing"

	"github.com/go-pg/pg"

	"github.com/insolar/observer/internal/testutils"
)

var db *pg.DB

type dbLogger struct{}

func (d dbLogger) BeforeQuery(q *pg.QueryEvent) {
}

func (d dbLogger) AfterQuery(q *pg.QueryEvent) {
}

func InitTestDB() func() {
	var dbCleaner func()
	db, _, dbCleaner = testutils.SetupDB("../../../../scripts/migrations")

	// for debug purposes print all queries
	db.AddQueryHook(dbLogger{})
	return dbCleaner
}

func TestMain(t *testing.M) {
	dbCleaner := InitTestDB()
	retCode := t.Run()
	dbCleaner()
	os.Exit(retCode)
}
