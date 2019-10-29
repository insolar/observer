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

package pg

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/ory/dockertest/v3"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer/store"
)

var _ store.RecordFetcher = (*Store)(nil)

var _ store.RecordSetter = (*Store)(nil)

var (
	db       *pg.DB
	database = "test_db"

	pgOptions = &pg.Options{
		Addr:            "localhost",
		User:            "postgres",
		Password:        "secret",
		Database:        "test_db",
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

	err = migrationCollection.DiscoverSQLMigrations("../../../../../scripts/migrations")
	if err != nil {
		log.Panicf("Failed to read migrations: %s", err)
	}

	_, _, err = migrationCollection.Run(db, "up")
	if err != nil {
		log.Panicf("Could not migrate: %s", err)
	}

	retCode := t.Run()

	err = pool.Purge(resource)
	if err != nil {
		log.Panicf("failed to purge docker pool: %s", err)
	}

	os.Exit(retCode)
}

func makeRequestWith(method string, reason insolar.Reference, args []byte) *record.Material {
	return &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{Union: &record.Virtual_IncomingRequest{IncomingRequest: &record.IncomingRequest{
			Method:    method,
			Reason:    reason,
			Arguments: args,
		}}}}
}

func TestStore_RequestGetSet(t *testing.T) {
	request := makeRequestWith("some", gen.RecordReference(), nil)
	store := NewPgStore(db)
	ctx := context.Background()
	require.NoError(t, store.SetRequest(ctx, *request))

	queryReq, err := store.Request(ctx, request.ID)
	require.NoError(t, err, "select request")
	require.Equal(t, *request, queryReq)
}

func TestStore_RequestBadRecord(t *testing.T) {
	request := makeSideEffectWith(gen.RecordReference(), nil)
	store := NewPgStore(db)
	ctx := context.Background()
	require.Error(t, store.SetRequest(ctx, *request))
}

func TestStore_DuplicateRequest(t *testing.T) {
	request := makeRequestWith("some", gen.RecordReference(), nil)
	store := NewPgStore(db)
	ctx := context.Background()
	require.NoError(t, store.SetRequest(ctx, *request))

	require.NoError(t, store.SetRequest(ctx, *request))
}

func TestStore_DuplicateResult(t *testing.T) {
	result := makeResultWith(gen.ID(), gen.RecordReference(), nil)
	store := NewPgStore(db)
	ctx := context.Background()
	require.NoError(t, store.SetResult(ctx, *result))

	require.NoError(t, store.SetResult(ctx, *result))
}

func TestStore_DuplicateSideEffect(t *testing.T) {
	sideEffect := makeSideEffectWith(gen.RecordReference(), nil)
	store := NewPgStore(db)
	ctx := context.Background()
	require.NoError(t, store.SetSideEffect(ctx, *sideEffect))

	require.NoError(t, store.SetSideEffect(ctx, *sideEffect))
}

func TestStore_RequestNotFound(t *testing.T) {
	pgStore := NewPgStore(db)
	queryReq, err := pgStore.Request(context.Background(), gen.ID())
	require.Equal(t, store.ErrNotFound, errors.Cause(err))
	require.Equal(t, record.Material{}, queryReq)
}

func makeResultWith(objID insolar.ID, request insolar.Reference, Payload []byte) *record.Material {
	return &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{Union: &record.Virtual_Result{Result: &record.Result{
			Object:  objID,
			Request: request,
			Payload: Payload,
		}}}}
}

func TestStore_ResultGetSet(t *testing.T) {
	reqRef := gen.RecordReference()
	result := makeResultWith(gen.ID(), reqRef, nil)
	store := NewPgStore(db)
	ctx := context.Background()
	require.NoError(t, store.SetResult(ctx, *result))

	queryRes, err := store.Result(ctx, *reqRef.GetLocal())
	require.NoError(t, err, "select result")
	require.Equal(t, *result, queryRes)
}

func makeSideEffectWith(request insolar.Reference, Payload []byte) *record.Material {
	return &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{Union: &record.Virtual_Amend{Amend: &record.Amend{
			Request: request,
			Memory:  Payload,
		}}}}
}

func TestStore_SideEffectGetSet(t *testing.T) {
	reqRef := gen.RecordReference()
	sideEffect := makeSideEffectWith(reqRef, nil)
	store := NewPgStore(db)
	ctx := context.Background()
	require.NoError(t, store.SetSideEffect(ctx, *sideEffect))

	querySideEffect, err := store.SideEffect(ctx, *reqRef.GetLocal())
	require.NoError(t, err, "select request")
	require.Equal(t, *sideEffect, querySideEffect)
}

func TestStore_CalledRequests(t *testing.T) {
	store := NewPgStore(db)
	ctx := context.Background()

	reasonRef := gen.RecordReference()

	requests := []record.Material{
		*makeRequestWith("one", reasonRef, nil),
		*makeRequestWith("two", reasonRef, nil),
	}

	notReasonedRequest := *makeRequestWith("wrong", gen.RecordReference(), nil)
	err := store.SetRequest(ctx, notReasonedRequest)
	require.NoError(t, err)

	for _, req := range requests {
		err := store.SetRequest(ctx, req)
		require.NoError(t, err)
	}

	actualRequests, err := store.CalledRequests(ctx, *reasonRef.GetLocal())
	require.NoError(t, err)
	require.Equal(t, requests, actualRequests)
}
