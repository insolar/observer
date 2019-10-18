package store

import (
	"context"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)

type CacheRecordStore struct {
	backend RecordStore
	cache *lru.Cache
}

func NewCacheRecordStore(backend RecordStore, size int) (*CacheRecordStore, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil,  errors.Wrap(err, "failed to init cache")
	}
	store := &CacheRecordStore{
		backend:backend,
		cache: cache,
	}
	return store, nil
}

type scope uint8

const (
	scopeRequest = iota
	scopeResult
	scopeSideEffect
)

type cacheKey struct {
	scope scope
	id insolar.ID
}

func (c *CacheRecordStore) SetRequest(ctx context.Context, record record.Material) error {
	err := c.backend.SetRequest(ctx, record)
	if err != nil {
		return err
	}
	return c.setCache(scopeRequest, record)
}

func (c *CacheRecordStore) SetResult(ctx context.Context, record record.Material) error {
	err := c.backend.SetResult(ctx, record)
	if err != nil {
		return err
	}
	return c.setCache(scopeResult, record)
}

func (c *CacheRecordStore) SetSideEffect(ctx context.Context, record record.Material) error {
	err := c.backend.SetSideEffect(ctx, record)
	if err != nil {
		return err
	}
	return c.setCache(scopeSideEffect, record)
}

func (c *CacheRecordStore) Request(ctx context.Context, reqID insolar.ID) (record.Material, error) {
	rec, ok := c.getCache(scopeRequest, reqID)
	if ok {
		return rec, nil
	}

	rec, err := c.backend.Request(ctx, reqID)
	if err != nil {
		return record.Material{}, err
	}
	err = c.setCache(scopeRequest, rec)
	if err != nil {
		return record.Material{}, errors.Wrap(err, "failed to write cache")
	}
	return rec, nil
}

func (c *CacheRecordStore) Result(ctx context.Context, reqID insolar.ID) (record.Material, error) {
	rec, ok := c.getCache(scopeResult, reqID)
	if ok {
		return rec, nil
	}

	rec, err := c.backend.Result(ctx, reqID)
	if err != nil {
		return record.Material{}, err
	}
	err = c.setCache(scopeResult, rec)
	if err != nil {
		return record.Material{}, errors.Wrap(err, "failed to write cache")
	}
	return rec, nil
}

func (c *CacheRecordStore) SideEffect(ctx context.Context, reqID insolar.ID) (record.Material, error) {
	rec, ok := c.getCache(scopeSideEffect, reqID)
	if ok {
		return rec, nil
	}

	rec, err := c.backend.SideEffect(ctx, reqID)
	if err != nil {
		return record.Material{}, err
	}
	err = c.setCache(scopeSideEffect, rec)
	if err != nil {
		return record.Material{}, errors.Wrap(err, "failed to write cache")
	}
	return rec, nil
}

func (c *CacheRecordStore) CalledRequests(ctx context.Context, reqID insolar.ID) ([]record.Material, error) {
	return c.backend.CalledRequests(ctx, reqID)
}

func (c *CacheRecordStore) setCache(sc scope, record record.Material) error {
	id, err := RequestID(&record)
	if err != nil {
		return errors.Wrap(err, "failed to extract request id")
	}
	_ = c.cache.Add(cacheKey{scope: sc, id: id}, record)
	return nil
}

func (c *CacheRecordStore) getCache(sc scope, reqID insolar.ID) (record.Material, bool) {
	val, ok := c.cache.Get(cacheKey{scope: sc, id: reqID})
	if !ok {
		return record.Material{}, false
	}
	rec, ok := val.(record.Material)
	if !ok {
		return record.Material{}, false
	}
	return rec, true
}
