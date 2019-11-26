package store

import (
	"context"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)

type CacheRecordStore struct {
	backend RecordStore

	requestLock sync.RWMutex
	cache       *lru.Cache
}

func NewCacheRecordStore(backend RecordStore, size int) (*CacheRecordStore, error) {
	store := &CacheRecordStore{backend: backend}
	cache, err := lru.New(size)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init cache")
	}
	store.cache = cache
	return store, nil
}

type scope uint8

const (
	scopeUnknown scope = iota // nolint
	scopeRequest
	scopeResult
	scopeSideEffect
	scopeCalledRequests
)

type cacheKey struct {
	scope scope
	id    insolar.ID
}

func (c *CacheRecordStore) SetRequest(ctx context.Context, record record.Material) error {
	reasonID, err := ReasonID(&record)
	if err != nil {
		return errors.Wrap(err, "failed to extract reason id")
	}

	c.requestLock.Lock()
	defer c.requestLock.Unlock()
	fromCache, ok := c.getMany(scopeCalledRequests, reasonID)
	if ok {
		// Found in cache, append required.
		fromCache = append(fromCache, &record)
	} else {
		// No append required because we already wrote the request to backend.
		fromBackend, err := c.backend.CalledRequests(ctx, reasonID)
		if err != nil {
			return err
		}
		fromCache = refMany(fromBackend)
	}

	c.setMany(scopeCalledRequests, reasonID, fromCache)
	return c.setOne(scopeRequest, &record)
}

func (c *CacheRecordStore) SetResult(ctx context.Context, record record.Material) error {
	return c.setOne(scopeResult, &record)
}

func (c *CacheRecordStore) SetSideEffect(ctx context.Context, record record.Material) error {
	return c.setOne(scopeSideEffect, &record)
}

func (c *CacheRecordStore) Request(ctx context.Context, reqID insolar.ID) (record.Material, error) {
	fromCache, ok := c.getOne(scopeRequest, reqID)
	if ok {
		return *fromCache, nil
	}

	fromBackend, err := c.backend.Request(ctx, reqID)
	if err != nil {
		return record.Material{}, err
	}
	err = c.setOne(scopeRequest, &fromBackend)
	if err != nil {
		return record.Material{}, errors.Wrap(err, "failed to write cache")
	}
	return fromBackend, nil
}

func (c *CacheRecordStore) Result(ctx context.Context, reqID insolar.ID) (record.Material, error) {
	fromCache, ok := c.getOne(scopeResult, reqID)
	if ok {
		return *fromCache, nil
	}

	fromBackend, err := c.backend.Result(ctx, reqID)
	if err != nil {
		return record.Material{}, err
	}
	err = c.setOne(scopeResult, &fromBackend)
	if err != nil {
		return record.Material{}, errors.Wrap(err, "failed to write cache")
	}
	return fromBackend, nil
}

func (c *CacheRecordStore) SideEffect(ctx context.Context, reqID insolar.ID) (record.Material, error) {
	fromCache, ok := c.getOne(scopeSideEffect, reqID)
	if ok {
		return *fromCache, nil
	}

	fromBackend, err := c.backend.SideEffect(ctx, reqID)
	if err != nil {
		return record.Material{}, err
	}
	err = c.setOne(scopeSideEffect, &fromBackend)
	if err != nil {
		return record.Material{}, errors.Wrap(err, "failed to write cache")
	}
	return fromBackend, nil
}

func (c *CacheRecordStore) CalledRequests(ctx context.Context, reqID insolar.ID) ([]record.Material, error) {
	c.requestLock.RLock()
	fromCache, ok := c.getMany(scopeCalledRequests, reqID)
	if ok {
		c.requestLock.RUnlock()
		return derefMany(fromCache), nil
	}
	c.requestLock.RUnlock()

	c.requestLock.Lock()
	defer c.requestLock.Unlock()
	fromCache, ok = c.getMany(scopeCalledRequests, reqID)
	if ok {
		return derefMany(fromCache), nil
	}

	fromBackend, err := c.backend.CalledRequests(ctx, reqID)
	if err != nil {
		return nil, err
	}
	c.setMany(scopeCalledRequests, reqID, refMany(fromBackend))
	return fromBackend, nil
}

func (c *CacheRecordStore) setOne(sc scope, record *record.Material) error {
	id, err := RequestID(record)
	if err != nil {
		return errors.Wrap(err, "failed to extract request id")
	}
	_ = c.cache.Add(cacheKey{scope: sc, id: id}, record)
	return nil
}

func (c *CacheRecordStore) getOne(sc scope, id insolar.ID) (*record.Material, bool) {
	val, ok := c.cache.Get(cacheKey{scope: sc, id: id})
	if !ok {
		return nil, false
	}
	rec, ok := val.(*record.Material)
	if !ok {
		return nil, false
	}
	return rec, true
}

func (c *CacheRecordStore) setMany(sc scope, id insolar.ID, records []*record.Material) {
	_ = c.cache.Add(cacheKey{scope: sc, id: id}, records)
}

func (c *CacheRecordStore) getMany(sc scope, id insolar.ID) ([]*record.Material, bool) {
	val, ok := c.cache.Get(cacheKey{scope: sc, id: id})
	if !ok {
		return nil, false
	}
	recs, ok := val.([]*record.Material)
	if !ok {
		return nil, false
	}
	return recs, true
}

func derefMany(in []*record.Material) []record.Material {
	out := make([]record.Material, len(in))
	for i, rec := range in {
		out[i] = *rec
	}
	return out
}

func refMany(in []record.Material) []*record.Material {
	out := make([]*record.Material, len(in))
	for i, rec := range in {
		rec := rec
		out[i] = &rec
	}
	return out
}
