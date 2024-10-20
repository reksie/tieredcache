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
	err = cache.Set(ctx, "example_key", "example_value", 5*time.Second)
	if err != nil {
		fmt.Printf("Error setting value: %v\n", err)
		return
	}

	value, err := cache.Get(ctx, "example_key")
	if err != nil {
		fmt.Printf("Error getting value: %v\n", err)
		return
	}

	fmt.Printf("Retrieved value: %v\n", value)

	// Wait for the value to expire
	time.Sleep(6 * time.Second)

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

	fetchFn := func(i int) (string, error) {
		atomic.AddInt32(&fetchCount, 1)
		return fmt.Sprintf("fetched_value_%d", atomic.LoadInt32(&fetchCount)), nil
	}

	// First call, should fetch
	i := 5
	result, err := tiercache.Swr(ctx, cache, []any{"swr_key", i}, fetchFn, tiercache.QueryOptions{Fresh: 2 * time.Second, TTL: 10 * time.Second})
	if err != nil {
		fmt.Printf("Error in SWR: %v\n", err)
		return
	}
	fmt.Printf("First call result: %v, Fetch count: %d\n", result, atomic.LoadInt32(&fetchCount))

	// Close the cache
	if err := cache.Close(); err != nil {
		fmt.Printf("Error closing cache: %v\n", err)
	}
}
