package tiercache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
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

func (tc *TieredCache) Swr(ctx context.Context, queryKey any, queryFn func(...any) (any, error), opts QueryOptions) (any, error) {
	return Swr(ctx, tc, queryKey, queryFn, opts)
}

func Swr[Props any, R any](ctx context.Context, tc *TieredCache, queryKey any, queryFn func(...Props) (R, error), opts QueryOptions) (R, error) {
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
		log.Printf("Swr: Found cached data for key: %s, type: %v", key, reflect.TypeOf(cachedData))

		cacheMap, ok := cachedData.(map[string]interface{})
		if !ok {
			log.Printf("Swr: Unexpected type for cached data: %T", cachedData)
			return zeroValue, errors.New("invalid cache item format")
		}

		log.Printf("Swr: Cache map contents: %+v", cacheMap)

		data, dataOk := cacheMap["data"]
		timestampStr, timeOk := cacheMap["timestamp"].(string)

		if !dataOk || !timeOk {
			log.Printf("Swr: Invalid cache item structure. Data ok: %v, Timestamp ok: %v", dataOk, timeOk)
			return zeroValue, errors.New("invalid cache item structure")
		}

		timestamp, err := time.Parse(time.RFC3339Nano, timestampStr)
		if err != nil {
			log.Printf("Swr: Error parsing timestamp: %v", err)
			return zeroValue, fmt.Errorf("error parsing timestamp: %v", err)
		}

		typedData, ok := data.(R)
		if !ok {
			log.Printf("Swr: Cannot convert cached data to required type. Expected %T, got %T", zeroValue, data)
			return zeroValue, errors.New("cannot convert cached data to required type")
		}

		age := time.Since(timestamp)
		log.Printf("Swr: Cache item age: %v, Fresh duration: %v", age, opts.Fresh)

		if age <= opts.Fresh {
			log.Printf("Swr: Returning fresh cached data for key: %s", key)
			return typedData, nil
		}

		log.Printf("Swr: Cached data is stale for key: %s, returning stale data and triggering refresh", key)
		go func() {
			log.Printf("Swr: Background refresh started for key: %s", key)
			newData, err := queryFn()
			if err == nil {
				log.Printf("Swr: Background refresh successful, updating cache for key: %s", key)
				tc.Set(ctx, key, CacheItem{Data: newData, Timestamp: time.Now()}, opts.TTL)
			} else {
				log.Printf("Swr: Background refresh failed for key: %s, error: %v", key, err)
			}
		}()

		return typedData, nil
	}

	log.Printf("Swr: No valid cached data found for key: %s, fetching new data", key)
	newData, err := queryFn()
	if err != nil {
		log.Printf("Swr: Error fetching new data for key: %s, error: %v", key, err)
		return zeroValue, err
	}

	log.Printf("Swr: Successfully fetched new data for key: %s, caching", key)
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
