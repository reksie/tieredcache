package tiercache

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	Data      any       `json:"data"`
	Timestamp time.Time `json:"timestamp"`
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

func (tc *TieredCache) Swr(ctx context.Context, queryKey any, queryFn func() (any, error), opts QueryOptions) (any, error) {
	return Swr(ctx, tc, queryKey, queryFn, opts)
}

func Swr[R any](ctx context.Context, tc *TieredCache, queryKey any, queryFn func() (R, error), opts QueryOptions) (R, error) {
	var zeroValue R

	key, err := generateKey(queryKey)
	if err != nil {
		return zeroValue, err
	}

	if opts.Fresh == 0 {
		opts.Fresh = tc.defaultFresh
	}

	log.Printf("Swr: Attempting to get key: %s", key)
	cachedData, err := tc.Get(ctx, key)
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
			newData, err := queryFn()
			if err == nil {
				tc.Set(ctx, key, CacheItem{Data: newData, Timestamp: time.Now()}, opts.TTL)
			} else {
				log.Printf("Swr: Background refresh failed for key: %s, error: %v", key, err)
			}
		}()

		return typedData, nil
	}

	newData, err := queryFn()
	if err != nil {
		return zeroValue, err
	}

	cacheItem := CacheItem{Data: newData, Timestamp: time.Now()}
	tc.Set(ctx, key, cacheItem, opts.TTL)

	return newData, nil
}

func generateKey(queryKey any) (string, error) {
	switch v := queryKey.(type) {
	case string:
		return keys.HashKeyMD5(v)
	case []any:
		key := ""
		for _, item := range v {
			key += fmt.Sprintf("%v:", item)
		}
		return keys.HashKeyMD5(key)
	default:
		return keys.HashKeyMD5(fmt.Sprintf("%v", queryKey))
	}
}
