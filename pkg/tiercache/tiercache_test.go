package tiercache

import (
	"context"
	"testing"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/reksie/memocache/pkg/interfaces"
	"github.com/reksie/memocache/pkg/stores"
	"github.com/stretchr/testify/assert"
)

var (
	cache *TieredCache
	ctx   context.Context
)

func TestMain(m *testing.M) {

	// Set up the memory cache
	bigcacheConfig := bigcache.DefaultConfig(10 * time.Minute)
	bigcacheInstance, _ := bigcache.New(context.Background(), bigcacheConfig)
	memoryStore := stores.CreateMemoryStore("memory", bigcacheInstance)

	cache = NewTieredCache(5*time.Second, []interfaces.CacheStore{memoryStore})
	ctx = context.Background()

	// Run the tests
	m.Run()

	// Clean up
	cache.Close()
}

func TestBasicSetAndGet(t *testing.T) {
	key := "test_key"
	value := "test_value"

	err := cache.Set(ctx, key, CacheItem{Data: value, Timestamp: time.Now()}, 100*time.Millisecond)
	assert.NoError(t, err)

	result, err := cache.Get(ctx, key)
	assert.NoError(t, err)

	cacheItem, ok := result.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, value, cacheItem["data"])
}

func TestExpiration(t *testing.T) {
	key := "expiring_key"
	value := "expiring_value"

	err := cache.Set(ctx, key, CacheItem{Data: value, Timestamp: time.Now()}, 50*time.Millisecond)
	assert.NoError(t, err)

	// Wait for the value to expire
	time.Sleep(100 * time.Millisecond)

	_, err = cache.Get(ctx, key)
	assert.Error(t, err)
}

func TestSWR(t *testing.T) {
	var fetchCount int

	queryFn := func() (string, error) {
		fetchCount++
		return "fetched_value", nil
	}

	// First call, should fetch
	result, err := Swr[string](QueryOptions{
		Context:       ctx,
		TieredCache:   cache,
		QueryKey:      []any{"swr_key"},
		QueryFunction: queryFn,
		Fresh:         50 * time.Millisecond,
		TTL:           200 * time.Millisecond,
	})

	assert.NoError(t, err)
	assert.Equal(t, "fetched_value", result)
	assert.Equal(t, 1, fetchCount)

	// Second call, should use cached value
	result, err = Swr[string](QueryOptions{
		Context:       ctx,
		TieredCache:   cache,
		QueryKey:      []any{"swr_key"},
		QueryFunction: queryFn,
		Fresh:         50 * time.Millisecond,
		TTL:           200 * time.Millisecond,
	})

	assert.NoError(t, err)
	assert.Equal(t, "fetched_value", result)
	assert.Equal(t, 1, fetchCount)

	// Wait for the fresh period to expire
	time.Sleep(100 * time.Millisecond)

	// Third call, should return stale data and trigger background refresh
	result, err = Swr[string](QueryOptions{
		Context:       ctx,
		TieredCache:   cache,
		QueryKey:      []any{"swr_key"},
		QueryFunction: queryFn,
		Fresh:         50 * time.Millisecond,
		TTL:           200 * time.Millisecond,
	})

	assert.NoError(t, err)
	assert.Equal(t, "fetched_value", result)

	// Wait for background refresh to complete
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 2, fetchCount)
}
