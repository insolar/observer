// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package pg

import (
	"context"
	"os"
	"testing"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/testutils"
)

var _ store.RecordFetcher = (*Store)(nil)

var _ store.RecordSetter = (*Store)(nil)

var db *pg.DB

func TestMain(t *testing.M) {
	var dbCleaner func()
	db, _, dbCleaner = testutils.SetupDB("../../../../../scripts/migrations")

	retCode := t.Run()
	dbCleaner()
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
	require.Equal(t, len(requests), len(actualRequests))
}
