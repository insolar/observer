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
	return
}

func (d dbLogger) AfterQuery(q *pg.QueryEvent) {
	return
}

func TestMain(t *testing.M) {
	var dbCleaner func()
	db, _, dbCleaner = testutils.SetupDB("../../../../scripts/migrations")

	// for debug purposes print all queries
	db.AddQueryHook(dbLogger{})

	retCode := t.Run()
	dbCleaner()
	os.Exit(retCode)
}
