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
// +build slowtest

package store

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	rand2 "math/rand"
	"os"
	"sort"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/insolar/insolar/insolar"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/require"

	"github.com/insolar/insolar/instrumentation/inslogger"
)

var (
	backend *sql.DB
)

var (
	testDB   = "test_db"
	testPort string
)

type testPostgresKey struct {
	id    []byte
	scope Scope
}

func (k testPostgresKey) Scope() Scope {
	return k.scope
}

func (k testPostgresKey) ID() []byte {
	return k.id
}

func TestMain(m *testing.M) {

	var (
		err error
	)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.Run("postgres", "11.4", []string{"POSTGRES_PASSWORD=secret", "POSTGRES_DB=" + testDB})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	if err = pool.Retry(func() error {
		var err error
		testPort = resource.GetPort("5432/tcp")
		backend, err = sql.Open("postgres", fmt.Sprintf("postgres://postgres:secret@localhost:%s/%s?sslmode=disable", testPort, testDB))
		if err != nil {
			return err
		}
		return backend.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	_, err = backend.Exec(`CREATE TABLE records(key TEXT, value TEXT, scope SMALLINT);`)
	if err != nil {
		log.Printf("failed to create table err: %v", err)
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestPostgresDB_SetGetSlow(t *testing.T) {
	var (
		ctx           = inslogger.TestContext(t)
		err           error
		key           testPostgresKey
		expectedValue []byte
	)
	//

	f := fuzz.New().NilChance(0)
	f.Fuzz(&key)
	f.Fuzz(&expectedValue)

	connStr := fmt.Sprintf("postgres://postgres:secret@localhost:%s/%s?sslmode=disable", testPort, testDB)
	db, err := NewPostgresDB(connStr)
	require.NoError(t, err)
	defer db.Stop(ctx)

	err = db.Set(key, expectedValue)
	require.NoError(t, err)

	value, err := db.Get(key)
	require.NoError(t, err)
	require.Equal(t, expectedValue, value)
}

func TestPostgresDB_NewIteratorSlow(t *testing.T) {
	type kv struct {
		k testPostgresKey
		v []byte
	}

	var (
		ctx          = inslogger.TestContext(t)
		commonScope  Scope
		commonPrefix []byte

		expected   []kv
		unexpected []kv
	)

	const (
		ArrayLength = 100
	)

	fuzz.New().NilChance(0).Fuzz(&commonScope)
	fuzz.New().NilChance(0).NumElements(ArrayLength, ArrayLength).Fuzz(&commonPrefix)

	f := fuzz.New().NilChance(0).NumElements(ArrayLength, ArrayLength).Funcs(
		func(key *testPostgresKey, c fuzz.Continue) {
			var id []byte
			c.Fuzz(&id)
			key.id = append(commonPrefix, id...)
			key.id[0] = commonPrefix[0] + 1
			key.scope = commonScope
		},
		func(pair *kv, c fuzz.Continue) {
			c.Fuzz(&pair.k)
			c.Fuzz(&pair.v)
		},
	)
	f.Fuzz(&unexpected)

	f = fuzz.New().NilChance(0).NumElements(ArrayLength, ArrayLength).Funcs(
		func(key *testPostgresKey, c fuzz.Continue) {
			var id []byte
			c.Fuzz(&id)
			key.id = append(commonPrefix, id...)
			key.scope = commonScope
		},
		func(pair *kv, c fuzz.Continue) {
			c.Fuzz(&pair.k)
			c.Fuzz(&pair.v)
		},
	)
	f.Fuzz(&expected)

	for _, pair := range unexpected {
		hexKey := hex.EncodeToString(pair.k.ID())
		hexValue := hex.EncodeToString(pair.v)
		backend.Exec(`INSERT INTO records(key, value, scope) VALUES($1, $2, $3);`, hexKey, hexValue, pair.k.Scope())
	}

	for _, pair := range expected {
		hexKey := hex.EncodeToString(pair.k.ID())
		hexValue := hex.EncodeToString(pair.v)
		backend.Exec(`INSERT INTO records(key, value, scope) VALUES($1, $2, $3);`, hexKey, hexValue, pair.k.Scope())
	}

	sort.Slice(expected, func(i, j int) bool {
		return bytes.Compare(expected[i].k.ID(), expected[j].k.ID()) == -1
	})

	connStr := fmt.Sprintf("postgres://postgres:secret@localhost:%s/%s?sslmode=disable", testPort, testDB)
	db, err := NewPostgresDB(connStr)
	require.NoError(t, err)
	defer db.Stop(ctx)

	// test logic
	pivot := testBadgerKey{id: commonPrefix, scope: commonScope}
	it := db.NewIterator(pivot, false)
	defer it.Close()
	i := 0
	for it.Next() && i < len(expected) {
		require.Equal(t, expected[i].k.ID(), it.Key(), "i: %v", i)
		val := it.Value()
		require.NoError(t, err, "i: %v", i)
		require.Equal(t, expected[i].v, val, "i: %v", i)
		i++
	}
	require.Equal(t, len(expected), i)
}

func TestPostgresDB_SimpleReverse(t *testing.T) {
	t.Parallel()

	var (
		ctx = inslogger.TestContext(t)
		err error
	)

	connStr := fmt.Sprintf("postgres://postgres:secret@localhost:%s/%s?sslmode=disable", testPort, testDB)
	db, err := NewPostgresDB(connStr)
	require.NoError(t, err)
	defer db.Stop(ctx)

	count := 100
	length := 10
	prefixes := make([][]byte, count)
	keys := make([][]byte, count)
	for i := 0; i < count; i++ {
		prefixes[i] = make([]byte, length)
		keys[i] = make([]byte, length)
		_, err = rand.Read(prefixes[i])
		require.NoError(t, err)
		_, err = rand.Read(keys[i])
		require.NoError(t, err)
		keys[i][0] = 0xFF
		keys[i] = append(prefixes[i], keys[i]...)
		err = db.Set(testBadgerKey{keys[i], ScopeRecord}, nil)
		require.NoError(t, err)
	}

	t.Run("ASC iteration", func(t *testing.T) {
		asc := make([][]byte, count)
		copy(asc, keys)
		sort.Slice(keys, func(i, j int) bool {
			return bytes.Compare(keys[i], keys[j]) == -1
		})
		sort.Slice(prefixes, func(i, j int) bool {
			return bytes.Compare(prefixes[i], prefixes[j]) == -1
		})

		seek := rand2.Intn(count)
		pivot := testPostgresKey{id: prefixes[seek], scope: ScopeRecord}
		it := db.NewIterator(pivot, false)
		defer it.Close()
		var actual [][]byte
		for it.Next() {
			actual = append(actual, it.Key())
		}
		require.Equal(t, count-seek, len(actual))
		require.Equal(t, keys[seek:], actual)
	})

	t.Run("DESC iteration", func(t *testing.T) {
		desc := make([][]byte, count)
		copy(desc, keys)
		sort.Slice(keys, func(i, j int) bool {
			return bytes.Compare(keys[i], keys[j]) >= 0
		})
		sort.Slice(prefixes, func(i, j int) bool {
			return bytes.Compare(prefixes[i], prefixes[j]) >= 0
		})

		seek := rand2.Intn(count)
		prefix := fillPrefix(prefixes[seek], length*2)
		pivot := testPostgresKey{id: prefix, scope: ScopeRecord}
		it := db.NewIterator(pivot, true)
		defer it.Close()
		var actual [][]byte
		for it.Next() {
			actual = append(actual, it.Key())
		}
		require.Equal(t, count-seek, len(actual))
		require.Equal(t, keys[seek:], actual)
	})
}

func TestPulse(t *testing.T) {
	t.Parallel()

	var (
		ctx = inslogger.TestContext(t)
		err error
	)

	connStr := `postgresql://localhost/yz?sslmode=disable`
	db, err := NewPostgresDB(connStr)
	require.NoError(t, err)
	defer db.Stop(ctx)

	val, err := db.Get(testPostgresKey{id: insolar.GenesisPulse.PulseNumber.Bytes(), scope: ScopePulse})
	// pulse := insolar.Pulse{}
	// insolar.Deserialize(val, &pulse)
	n := dbNode{}
	insolar.Deserialize(val, &n)
	t.Logf("pulse: %v %v %v", n.Pulse, n.Prev, n.Next)
}

type dbNode struct {
	Pulse      insolar.Pulse
	Prev, Next *insolar.PulseNumber
}
