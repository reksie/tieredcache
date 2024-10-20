package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/reksie/memocache/pkg/stores"
	"github.com/reksie/memocache/pkg/tiercache"
)

func main() {
	// Create a BigCache instance
	bigcacheConfig := bigcache.DefaultConfig(10 * time.Minute)
	bigcacheInstance, err := bigcache.New(context.Background(), bigcacheConfig)
	if err != nil {
		fmt.Printf("Error creating BigCache: %v\n", err)
		return
	}

	// Create a new tiercache with a memory store
	memoryStore := stores.CreateMemoryStore("memory", bigcacheInstance)
	cache := tiercache.NewTieredCache(5*time.Second, memoryStore)

	ctx := context.Background()

	// Basic set and get test
	fmt.Println("Basic Set and Get Test:")
	err = cache.Set(ctx, "example_key", tiercache.CacheItem{Data: "example_value", Timestamp: time.Now()}, 100*time.Millisecond)
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
	var fetchCount int32

	i := 9

	queryFn := func() (string, error) {
		fetchCount++
		return fmt.Sprintf("fetched_value_%d_%d", i, atomic.LoadInt32(&fetchCount)), nil
	}

	// First call, should fetch
	result, err := tiercache.Swr[string](tiercache.QueryOptions{
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
	result, err = tiercache.Swr[string](tiercache.QueryOptions{
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
	result, err = tiercache.Swr[string](tiercache.QueryOptions{
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
	result, err = tiercache.Swr[string](tiercache.QueryOptions{
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
