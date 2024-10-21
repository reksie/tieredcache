package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/redis/go-redis/v9"
	"github.com/reksie/tieredcache/pkg/interfaces"
	"github.com/reksie/tieredcache/pkg/keys"
	"github.com/reksie/tieredcache/pkg/stores"
	"github.com/reksie/tieredcache/pkg/tieredcache"
)

func main() {

	ctx := context.Background()

	// Create a BigCache instance
	bigcacheConfig := bigcache.DefaultConfig(10 * time.Minute)
	bigcacheInstance, err := bigcache.New(context.Background(), bigcacheConfig)
	if err != nil {
		fmt.Printf("Error creating BigCache: %v\n", err)
		return
	}

	// Create a new tiercache with a memory store
	memoryStore := stores.CreateMemoryStore("memory", bigcacheInstance)

	// setup the redis cache

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	redisStore := stores.CreateRedisStore("redis", redisClient, stores.RedisStoreConfig{
		UseJSONMarshalling: true,
		UseIntegerForTTL:   true,
	})

	cache := tieredcache.NewTieredCache(5*time.Second, []interfaces.CacheStore{memoryStore, redisStore})

	type ExampleParams struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	exampleParams := ExampleParams{Name: "John", Age: 30}

	key, err := keys.HashKeyMD5(exampleParams)
	if err != nil {
		fmt.Printf("Error hashing key: %v\n", err)
		return
	}

	// Then use the Swr
	var fetchCount int32
	i := 9

	queryFn := func() (string, error) {
		fetchCount++
		return fmt.Sprintf("fetched_value_%d_%d", i, atomic.LoadInt32(&fetchCount)), nil
	}

	// Building own query key
	result, err := tieredcache.Swr[string](tieredcache.QueryOptions[string]{
		Context:       ctx,
		TieredCache:   cache,
		QueryKey:      []string{"examplePrefix", key},
		QueryFunction: queryFn,
		Fresh:         2 * time.Second,
		TTL:           5 * time.Minute,
	})
	if err != nil {
		fmt.Printf("Error in SWR: %v\n", err)
	}

	fmt.Printf("First call result: %v\n", result)

	// Basic set and get test
	fmt.Println("Basic Set and Get Test:")
	err = cache.Set(ctx, "example_key", tieredcache.CacheItem{Data: "example_value", Timestamp: time.Now()}, 100*time.Millisecond)
	if err != nil {
		fmt.Printf("Error setting value: %v\n", err)
		return
	}

	value, err := cache.Get(ctx, "example_key")
	if err != nil {
		fmt.Printf("Error getting value: %v\n", err)
		return
	}

	if cacheItem, ok := value.(map[string]interface{}); ok {
		fmt.Printf("Retrieved value: %v\n", cacheItem["data"])
	} else {
		fmt.Printf("Unexpected type for retrieved value: %T\n", value)
	}

	// Wait for the value to expire
	time.Sleep(200 * time.Millisecond)

	// Try to get the expired value
	value, err = cache.Get(ctx, "example_key")
	if err != nil {
		fmt.Printf("Expected error after expiration: %v\n", err)
	} else {
		fmt.Printf("Unexpected: value still exists after expiration: %v\n", value)
	}

	// SWR Test
	fmt.Println("\nSWR Test:")

	// First call, should fetch
	result, err = tieredcache.Swr[string](tieredcache.QueryOptions[string]{
		Context:       ctx,
		TieredCache:   cache,
		QueryKey:      []any{"swr_key", i},
		QueryFunction: queryFn,
		Fresh:         2 * time.Second,
		TTL:           10 * time.Second,
	})
	if err != nil {
		fmt.Printf("Error in SWR: %v\n", err)
		return
	}
	fmt.Printf("First call result: %v, Fetch count: %d\n", result, atomic.LoadInt32(&fetchCount))

	// Second call, should use cached value
	result, err = tieredcache.Swr[string](tieredcache.QueryOptions[string]{
		Context:       ctx,
		TieredCache:   cache,
		QueryKey:      []any{"swr_key", i},
		QueryFunction: queryFn,
		Fresh:         1 * time.Second,
		TTL:           10 * time.Second,
	})
	if err != nil {
		fmt.Printf("Error in second SWR call: %v\n", err)
		return
	}
	fmt.Printf("Second call result: %v, Fetch count: %d\n", result, atomic.LoadInt32(&fetchCount))

	// Wait for the fresh period to expire
	time.Sleep(2 * time.Second)

	// Third call, should return stale data and trigger background refresh
	result, err = tieredcache.Swr[string](tieredcache.QueryOptions[string]{
		Context:       ctx,
		TieredCache:   cache,
		QueryKey:      []any{"swr_key", i},
		QueryFunction: queryFn,
		Fresh:         1 * time.Second,
		TTL:           10 * time.Second,
	})
	if err != nil {
		fmt.Printf("Error in third SWR call: %v\n", err)
		return
	}
	fmt.Printf("Third call result: %v, Fetch count: %d\n", result, atomic.LoadInt32(&fetchCount))

	// Wait for background refresh to complete
	time.Sleep(100 * time.Millisecond)

	// Fourth call, should return the newly refreshed data
	result, err = tieredcache.Swr[string](tieredcache.QueryOptions[string]{
		Context:       ctx,
		TieredCache:   cache,
		QueryKey:      []any{"swr_key", i},
		QueryFunction: queryFn,
		Fresh:         2 * time.Second,
		TTL:           10 * time.Second,
	})
	if err != nil {
		fmt.Printf("Error in fourth SWR call: %v\n", err)
		return
	}
	fmt.Printf("Fourth call result: %v, Fetch count: %d\n", result, atomic.LoadInt32(&fetchCount))

	// Close the cache
	if err := cache.Close(); err != nil {
		fmt.Printf("Error closing cache: %v\n", err)
	}
}
