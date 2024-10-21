# Introduction

This is an experimental go multi tiered cache implementation.

## Usage

```go
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

	redisStore := stores.CreateRedisStore("test_store", redisClient, stores.RedisStoreConfig{
		UseJSONMarshalling: true,
		UseIntegerForTTL:   true,
	})

	cache := tieredcache.NewTieredCache(5*time.Second, []interfaces.CacheStore{memoryStore, redisStore})

	// Then use the Swr
	var fetchCount int32
	i := 9

	queryFn := func() (string, error) {
		fetchCount++
		return fmt.Sprintf("fetched_value_%d_%d", i, atomic.LoadInt32(&fetchCount)), nil
	}


	type ExampleParams struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	exampleParams := ExampleParams{Name: "John", Age: 30}

	key, err := keys.HashKeyMD5(exampleParams) // or keys.HashKeyJSON(exampleParams)
	if err != nil {
		fmt.Printf("Error hashing key: %v\n", err)
		return
	}

	// Building own query key
	result, err := tieredcache.Swr[string](tieredcache.QueryOptions{ // [string] is the return type of the query function
		Context:       ctx,
		TieredCache:   cache,
		QueryKey:      []string{"examplePrefix", key},
		QueryFunction: queryFn,
		Fresh:         2 * time.Second,
		TTL:           5 * time.Minute,
	})
}
```

## Improvements

- in the case we have multiple calls we can try to use something like [singleflight](https://pkg.go.dev/golang.org/x/sync@v0.8.0/singleflight) to try and dedupe.
- however within a request unless there's multiple paralell calls, we will typically be waiting when there is a cache miss, and future requests will get cache hits.
- improve use of generics and revisit the interface for a `QueryKey`
