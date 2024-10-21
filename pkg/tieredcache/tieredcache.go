package tieredcache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/reksie/tieredcache/pkg/interfaces"
	"github.com/reksie/tieredcache/pkg/keys"
)

type QueryFunction[R any] func() (R, error)

type QueryOptions[R any] struct {
	Context       context.Context
	TieredCache   *TieredCache
	QueryKey      any
	QueryFunction QueryFunction[R]
	Fresh         time.Duration
	TTL           time.Duration
}

type QueryResult struct {
	Data  any
	Error error
}

type CacheItem struct {
	Data      any       `json:"data"`
	Timestamp time.Time `json:"timestamp"` // maybe more optimal to store as Unix timestamp
}

type TieredCache struct {
	stores       []interfaces.CacheStore
	defaultFresh time.Duration
}

func NewTieredCache(defaultFresh time.Duration, stores []interfaces.CacheStore) *TieredCache {
	return &TieredCache{
		stores:       stores,
		defaultFresh: defaultFresh,
	}
}

func (tc *TieredCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	cacheItem, ok := value.(CacheItem)
	if !ok {
		return errors.New("value must be a CacheItem")
	}

	storeValue := map[string]interface{}{
		"data":      cacheItem.Data,
		"timestamp": cacheItem.Timestamp.Format(time.RFC3339Nano),
	}

	for _, store := range tc.stores {
		if err := store.Set(ctx, key, storeValue, ttl); err != nil {
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

func Swr[R any](opts QueryOptions[R]) (R, error) {
	var zeroValue R

	key, err := generateKey(opts.QueryKey)
	if err != nil {
		return zeroValue, err
	}

	if opts.Fresh == 0 {
		opts.Fresh = opts.TieredCache.defaultFresh
	}

	cachedData, err := opts.TieredCache.Get(opts.Context, key)
	if err == nil && cachedData != nil {
		cacheMap, ok := cachedData.(map[string]interface{})
		if !ok {
			return zeroValue, errors.New("invalid cache item format")
		}

		data, dataOk := cacheMap["data"]
		timestampStr, timeOk := cacheMap["timestamp"].(string)

		if !dataOk || !timeOk {
			return zeroValue, errors.New("invalid cache item structure")
		}

		timestamp, err := time.Parse(time.RFC3339Nano, timestampStr)
		if err != nil {
			return zeroValue, fmt.Errorf("error parsing timestamp: %v", err)
		}

		typedData, ok := data.(R)
		if !ok {
			return zeroValue, errors.New("cannot convert cached data to required type")
		}

		age := time.Since(timestamp)

		if age <= opts.Fresh {
			return typedData, nil
		}

		go func() {

			newData, err := opts.QueryFunction()
			if err == nil {
				opts.TieredCache.Set(opts.Context, key, CacheItem{Data: newData, Timestamp: time.Now()}, opts.TTL)
			} else {
				log.Printf("Swr: Background refresh failed for key: %s, error: %v", key, err)
			}
		}()

		return typedData, nil
	}

	newData, err := opts.QueryFunction()
	if err != nil {
		return zeroValue, err
	}

	cacheItem := CacheItem{Data: newData, Timestamp: time.Now()}
	opts.TieredCache.Set(opts.Context, key, cacheItem, opts.TTL)

	return newData, nil
}

func generateKey(queryKey any) (string, error) {

	// keys.HashKeyMD5
	hashFunction := keys.HashKeyJson

	switch v := queryKey.(type) {
	case string:
		return hashFunction(v)
	case []any:
		key := ""
		for _, item := range v {
			key += fmt.Sprintf("%v:", item)
		}
		return hashFunction(key)
	default:
		return hashFunction(fmt.Sprintf("%v", queryKey))
	}
}
