package stores

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/reksie/memocache/pkg/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

var (
	redisClient *redis.Client
	cleanup     func()
)

func TestMain(m *testing.M) {
	// Setup
	ctx := context.Background()
	redisContainer, err := rediscontainer.RunContainer(ctx,
		testcontainers.WithImage("redis:6-alpine"),
	)
	if err != nil {
		panic(err)
	}

	endpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		panic(err)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: endpoint,
	})

	cleanup = func() {
		redisClient.Close()
		if err := redisContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	}

	// Run tests
	code := m.Run()

	// Teardown
	cleanup()

	os.Exit(code)
}

func setupTest(t *testing.T) interfaces.CacheStore {
	store := CreateRedisStore("test_store", redisClient)
	err := store.Clear() // Clear the store before each test
	assert.NoError(t, err)
	return store
}

func TestRedisCreateStore(t *testing.T) {
	store := setupTest(t)
	assert.Equal(t, "test_store", store.Name())
}

func TestRedisSet(t *testing.T) {
	store := setupTest(t)
	err := store.Set("key1", "value1", 60*time.Second)
	assert.NoError(t, err)
}

func TestRedisGet(t *testing.T) {
	store := setupTest(t)
	err := store.Set("key1", "value1", 60*time.Second)
	assert.NoError(t, err)

	value, err := store.Get("key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", value)

	_, err = store.Get("non_existent_key")
	assert.Error(t, err)
}

func TestRedisGetExpired(t *testing.T) {
	store := setupTest(t)
	err := store.Set("key1", "value1", 10*time.Millisecond)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	val, err := store.Get("key1")
	fmt.Println(val)
	assert.Error(t, err)
}

func TestRedisDelete(t *testing.T) {
	store := setupTest(t)
	err := store.Set("key1", "value1", 60*time.Second)
	assert.NoError(t, err)

	err = store.Delete("key1")
	assert.NoError(t, err)

	_, err = store.Get("key1")
	assert.Error(t, err)
}

func TestRedisClear(t *testing.T) {
	store := setupTest(t)
	err := store.Set("key1", "value1", 60*time.Second)
	assert.NoError(t, err)
	err = store.Set("key2", "value2", 60*time.Second)
	assert.NoError(t, err)

	err = store.Clear()
	assert.NoError(t, err)

	_, err1 := store.Get("key1")
	_, err2 := store.Get("key2")
	assert.Error(t, err1)
	assert.Error(t, err2)
}

func TestRedisClose(t *testing.T) {
	store := setupTest(t)
	err := store.Close()
	assert.NoError(t, err)
}