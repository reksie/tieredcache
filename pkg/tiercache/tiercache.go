package tiercache

import (
	"context"
	"errors"
	"time"

	"github.com/reksie/memocache/pkg/interfaces"
	"github.com/reksie/memocache/pkg/keys"
)

type QueryOptions struct {
	Fresh time.Duration
	TTL   time.Duration
}

type QueryResult struct {
	Data  any
	Error error
}

type CacheItem struct {
	Data      any
	Timestamp time.Time
}

type TieredCache struct {
	stores       []interfaces.CacheStore
	defaultFresh time.Duration
}

func NewTieredCache(defaultFresh time.Duration, stores ...interfaces.CacheStore) *TieredCache {
	return &TieredCache{
		stores:       stores,
		defaultFresh: defaultFresh,
	}
}

func (tc *TieredCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	for _, store := range tc.stores {
		if err := store.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TieredCache) Get(ctx context.Context, key string) (any, error) {
	for _, store := range tc.stores {
		if value, err := store.Get(ctx, key); err == nil {
			return value, nil
		}
	}
	return nil, errors.New("key not found in any store")
}

func (tc *TieredCache) Delete(ctx context.Context, key string) error {
	for _, store := range tc.stores {
		if err := store.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TieredCache) Clear(ctx context.Context) error {
	for _, store := range tc.stores {
		if err := store.Clear(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TieredCache) Close() error {
	for _, store := range tc.stores {
		if err := store.Close(); err != nil {
			return err
		}
	}
	return nil
}

// We can't make this a method of TieredCache because we'd like the generic type R to be inferred, however we can make a tieredcache any and leave it up to the user to pass in the correct type

func (tc *TieredCache) Swr(ctx context.Context, queryKey any, queryFn func(...any) (any, error), opts QueryOptions) (any, error) {
	return Swr(ctx, tc, queryKey, queryFn, opts)
}

func Swr[Props any, R any](ctx context.Context, tc *TieredCache, queryKey any, queryFn func(...Props) (R, error), opts QueryOptions) (R, error) {

	var zeroValue R

	key, error := generateKey(queryKey)
	if error != nil {
		return zeroValue, error
	}

	// If Fresh is not set, use the default
	if opts.Fresh == 0 {
		opts.Fresh = tc.defaultFresh
	}

	// Try to get from cache

	cachedData, err := tc.Get(ctx, key)
	if err == nil {
		cacheItem := cachedData.(CacheItem)
		// Data found in cache
		go func() {
			// Asynchronously check if data is stale and needs revalidation
			if time.Since(cacheItem.Timestamp) > opts.Fresh {
				newData, err := queryFn()
				if err == nil {
					tc.Set(ctx, key, CacheItem{Data: newData, Timestamp: time.Now()}, opts.TTL)
				}
			}
		}()
		return cacheItem.Data.(R), nil
	}

	// Data not found in cache or error occurred, fetch new data
	newData, err := queryFn()
	if err != nil {
		return zeroValue, err
	}

	// Store new data in cache
	cacheItem := CacheItem{Data: newData, Timestamp: time.Now()}
	tc.Set(ctx, key, cacheItem, opts.TTL)

	return newData, nil
}

func generateKey(queryKey any) (string, error) {
	return keys.HashKeyMD5(queryKey)
}
