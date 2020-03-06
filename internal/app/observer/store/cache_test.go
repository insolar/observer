// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package store

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheRecordStore_Request(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	cacheSize := 10

	var (
		backend *RecordStoreMock
		cache   *CacheRecordStore
	)
	setup := func() {
		backend = NewRecordStoreMock(mc)
		c, err := NewCacheRecordStore(backend, cacheSize)
		if err != nil {
			panic(err)
		}
		cache = c
	}
	genRecord := func() record.Material {
		return record.Material{ID: gen.ID(), Virtual: record.Wrap(&record.IncomingRequest{})}
	}

	expectedRecord := genRecord()
	expectedRequestID, err := RequestID(&expectedRecord)
	require.NoError(t, err)

	t.Run("not found in backend", func(t *testing.T) {
		setup()
		defer mc.Finish()

		backend.RequestMock.Inspect(func(ctx context.Context, reqID insolar.ID) {
			require.Equal(t, expectedRequestID, reqID)
		}).Return(record.Material{}, ErrNotFound)

		_, err := cache.Request(ctx, expectedRequestID)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("found in backend", func(t *testing.T) {
		setup()
		defer mc.Finish()

		backend.RequestMock.Inspect(func(ctx context.Context, reqID insolar.ID) {
			require.Equal(t, expectedRecord.ID, reqID)
		}).Return(expectedRecord, nil)

		rec, err := cache.Request(ctx, expectedRequestID)
		assert.NoError(t, err)
		assert.Equal(t, expectedRecord, rec)

		rec, err = cache.Request(ctx, expectedRequestID)
		assert.NoError(t, err)
		assert.Equal(t, expectedRecord, rec)
		assert.Equal(t, 1, len(backend.RequestMock.Calls()), "should not call backend on second read")
	})

	t.Run("sets in cache", func(t *testing.T) {
		setup()
		defer mc.Finish()

		err := cache.SetRequest(ctx, expectedRecord)
		assert.NoError(t, err)

		rec, err := cache.Request(ctx, expectedRequestID)
		assert.NoError(t, err)
		assert.Equal(t, expectedRecord, rec)
	})

	t.Run("evicts the last record", func(t *testing.T) {
		setup()
		defer mc.Finish()

		backend.RequestMock.Return(record.Material{}, ErrNotFound)

		// Set expected (will be the last).
		err := cache.SetRequest(ctx, expectedRecord)
		assert.NoError(t, err)

		// Fill the cache until it overflows.
		for i := 0; i < cacheSize; i++ {
			err := cache.SetRequest(ctx, genRecord())
			assert.NoError(t, err)
		}

		// Expected record evicted.
		_, err = cache.Request(ctx, expectedRequestID)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("updated usage for accessed record", func(t *testing.T) {
		setup()
		defer mc.Finish()

		// Set expected (will be the last).
		err := cache.SetRequest(ctx, expectedRecord)
		assert.NoError(t, err)

		// Fill half the cache.
		for i := 0; i < cacheSize/2; i++ {
			err := cache.SetRequest(ctx, genRecord())
			assert.NoError(t, err)
		}

		// Access expected record and update its last use.
		_, err = cache.Request(ctx, expectedRequestID)
		assert.NoError(t, err)

		// Fill another half.
		for i := 0; i < cacheSize/2; i++ {
			err := cache.SetRequest(ctx, genRecord())
			assert.NoError(t, err)
		}

		// Expected record is not evicted.
		_, err = cache.Request(ctx, expectedRequestID)
		assert.NoError(t, err)
	})
}

func TestCacheRecordStore_Result(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	cacheSize := 10

	var (
		backend *RecordStoreMock
		cache   *CacheRecordStore
	)
	setup := func() {
		backend = NewRecordStoreMock(mc)
		c, err := NewCacheRecordStore(backend, cacheSize)
		if err != nil {
			panic(err)
		}
		cache = c
	}
	genRecord := func() record.Material {
		return record.Material{Virtual: record.Wrap(&record.Result{Request: gen.Reference()})}
	}

	expectedRecord := genRecord()
	expectedRequestID, err := RequestID(&expectedRecord)
	require.NoError(t, err)

	t.Run("not found in backend", func(t *testing.T) {
		setup()
		defer mc.Finish()

		backend.ResultMock.Inspect(func(ctx context.Context, reqID insolar.ID) {
			require.Equal(t, expectedRequestID, reqID)
		}).Return(record.Material{}, ErrNotFound)

		_, err := cache.Result(ctx, expectedRequestID)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("found in backend", func(t *testing.T) {
		setup()
		defer mc.Finish()

		backend.ResultMock.Inspect(func(ctx context.Context, reqID insolar.ID) {
			require.Equal(t, expectedRequestID, reqID)
		}).Return(expectedRecord, nil)

		rec, err := cache.Result(ctx, expectedRequestID)
		assert.NoError(t, err)
		assert.Equal(t, expectedRecord, rec)

		rec, err = cache.Result(ctx, expectedRequestID)
		assert.NoError(t, err)
		assert.Equal(t, expectedRecord, rec)
		assert.Equal(t, 1, len(backend.ResultMock.Calls()), "should not call backend on second read")
	})

	t.Run("sets in cache", func(t *testing.T) {
		setup()
		defer mc.Finish()

		err := cache.SetResult(ctx, expectedRecord)
		assert.NoError(t, err)

		rec, err := cache.Result(ctx, expectedRequestID)
		assert.NoError(t, err)
		assert.Equal(t, expectedRecord, rec)
	})

	t.Run("evicts the last record", func(t *testing.T) {
		setup()
		defer mc.Finish()

		backend.ResultMock.Return(record.Material{}, ErrNotFound)

		// Set expected (will be the last).
		err := cache.SetResult(ctx, expectedRecord)
		assert.NoError(t, err)

		// Fill the cache until it overflows.
		for i := 0; i < cacheSize; i++ {
			err := cache.SetResult(ctx, genRecord())
			assert.NoError(t, err)
		}

		// Expected record evicted.
		_, err = cache.Result(ctx, expectedRequestID)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("updated usage for accessed record", func(t *testing.T) {
		setup()
		defer mc.Finish()

		// Set expected (will be the last).
		err := cache.SetResult(ctx, expectedRecord)
		assert.NoError(t, err)

		// Fill the cache until full.
		for i := 0; i < cacheSize-1; i++ {
			err := cache.SetResult(ctx, genRecord())
			assert.NoError(t, err)
		}

		// Access expected record and update its last use.
		_, err = cache.Result(ctx, expectedRequestID)
		assert.NoError(t, err)

		// Fill one more record so the last one is evicted.
		err = cache.SetResult(ctx, genRecord())
		assert.NoError(t, err)

		// Expected record is not evicted.
		_, err = cache.Result(ctx, expectedRequestID)
		assert.NoError(t, err)
	})
}

func TestCacheRecordStore_SideEffect(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	cacheSize := 10

	var (
		backend *RecordStoreMock
		cache   *CacheRecordStore
	)
	setup := func() {
		backend = NewRecordStoreMock(mc)
		c, err := NewCacheRecordStore(backend, cacheSize)
		if err != nil {
			panic(err)
		}
		cache = c
	}
	genRecord := func() record.Material {
		return record.Material{Virtual: record.Wrap(&record.Amend{Request: gen.Reference()})}
	}

	expectedRecord := genRecord()
	expectedRequestID, err := RequestID(&expectedRecord)
	require.NoError(t, err)

	t.Run("not found in backend", func(t *testing.T) {
		setup()
		defer mc.Finish()

		backend.SideEffectMock.Inspect(func(ctx context.Context, reqID insolar.ID) {
			require.Equal(t, expectedRequestID, reqID)
		}).Return(record.Material{}, ErrNotFound)

		_, err := cache.SideEffect(ctx, expectedRequestID)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("found in backend", func(t *testing.T) {
		setup()
		defer mc.Finish()

		backend.SideEffectMock.Inspect(func(ctx context.Context, reqID insolar.ID) {
			require.Equal(t, expectedRequestID, reqID)
		}).Return(expectedRecord, nil)

		rec, err := cache.SideEffect(ctx, expectedRequestID)
		assert.NoError(t, err)
		assert.Equal(t, expectedRecord, rec)

		rec, err = cache.SideEffect(ctx, expectedRequestID)
		assert.NoError(t, err)
		assert.Equal(t, expectedRecord, rec)
		assert.Equal(t, 1, len(backend.SideEffectMock.Calls()), "should not call backend on second read")
	})

	t.Run("sets in cache", func(t *testing.T) {
		setup()
		defer mc.Finish()

		err := cache.SetSideEffect(ctx, expectedRecord)
		assert.NoError(t, err)

		rec, err := cache.SideEffect(ctx, expectedRequestID)
		assert.NoError(t, err)
		assert.Equal(t, expectedRecord, rec)
	})

	t.Run("evicts the last record", func(t *testing.T) {
		setup()
		defer mc.Finish()

		backend.SideEffectMock.Return(record.Material{}, ErrNotFound)

		// Set expected (will be the last).
		err := cache.SetSideEffect(ctx, expectedRecord)
		assert.NoError(t, err)

		// Fill the cache until it overflows.
		for i := 0; i < cacheSize; i++ {
			err := cache.SetSideEffect(ctx, genRecord())
			assert.NoError(t, err)
		}

		// Expected record evicted.
		_, err = cache.SideEffect(ctx, expectedRequestID)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("updated usage for accessed record", func(t *testing.T) {
		setup()
		defer mc.Finish()

		// Set expected (will be the last).
		err := cache.SetSideEffect(ctx, expectedRecord)
		assert.NoError(t, err)

		// Fill the cache until full.
		for i := 0; i < cacheSize-1; i++ {
			err := cache.SetSideEffect(ctx, genRecord())
			assert.NoError(t, err)
		}

		// Access expected record and update its last use.
		_, err = cache.SideEffect(ctx, expectedRequestID)
		assert.NoError(t, err)

		// Fill one more record so the last one is evicted.
		err = cache.SetSideEffect(ctx, genRecord())
		assert.NoError(t, err)

		// Expected record is not evicted.
		_, err = cache.SideEffect(ctx, expectedRequestID)
		assert.NoError(t, err)
	})
}

func TestCacheRecordStore_CalledRequests(t *testing.T) {
	ctx := context.Background()
	mc := minimock.NewController(t)
	cacheSize := 10

	var (
		backend *RecordStoreMock
		cache   *CacheRecordStore
	)
	setup := func() {
		backend = NewRecordStoreMock(mc)
		c, err := NewCacheRecordStore(backend, cacheSize)
		if err != nil {
			panic(err)
		}
		cache = c
	}
	genRecord := func() record.Material {
		return record.Material{ID: gen.ID(), Virtual: record.Wrap(&record.IncomingRequest{})}
	}

	requestRecord := genRecord()
	expectedReasonID, err := ReasonID(&requestRecord)
	require.NoError(t, err)

	t.Run("sets in cache", func(t *testing.T) {
		setup()
		defer mc.Finish()

		backend.CalledRequestsMock.Set(func(ctx context.Context, reqID insolar.ID) (ma1 []record.Material, err error) {
			require.Equal(t, expectedReasonID, reqID)
			return []record.Material{requestRecord}, nil
		})

		err := cache.SetRequest(ctx, requestRecord)
		assert.NoError(t, err)

		rec, err := cache.CalledRequests(ctx, expectedReasonID)
		assert.NoError(t, err)
		assert.Equal(t, []record.Material{requestRecord}, rec)
	})
}
