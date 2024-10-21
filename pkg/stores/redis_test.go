package stores

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/redis/go-redis/v9"
	"github.com/reksie/memocache/pkg/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	redisClient *redis.Client
	cleanup     func()
)

const localTest = false

func TestMain(m *testing.M) {
	// Setup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log.Println("Setting up test environment...")

	req := testcontainers.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		log.Fatalf("Failed to start Redis container: %v", err)
	}

	endpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		log.Fatalf("Failed to get Redis endpoint: %v", err)
	}

	if localTest {
		endpoint = "localhost:6379"
	}

	log.Printf("Redis endpoint: %s", endpoint)

	redisClient = redis.NewClient(&redis.Options{
		Addr: endpoint,
	})

	// Run tests
	code := m.Run()

	// Teardown
	log.Println("Cleaning up test environment...")
	redisClient.Close()
	if err := redisContainer.Terminate(ctx); err != nil {
		log.Printf("Failed to terminate Redis container: %v", err)
	}

	os.Exit(code)
}

func setupTest(t *testing.T) interfaces.CacheStore {
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: false,
		UseIntegerForTTL:   true,
	})
	ctx := context.Background()
	err := store.Clear(ctx) // Clear the store before each test
	assert.NoError(t, err)
	return store
}

func TestRedisCreateStore(t *testing.T) {
	store := setupTest(t)
	assert.Equal(t, "test_store", store.Name())
}

func TestRedisSet(t *testing.T) {
	ctx := context.Background()
	store := setupTest(t)
	err := store.Set(ctx, "key1", "value1", 60*time.Second)
	assert.NoError(t, err)
}

func TestRedisGet(t *testing.T) {
	ctx := context.Background()
	store := setupTest(t)

	err := store.Set(ctx, "key1", "value1", 60*time.Second)
	assert.NoError(t, err)

	value, err := store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", value)

	_, err = store.Get(ctx, "non_existent_key")
	assert.Error(t, err)
}

func TestRedisGetExpired(t *testing.T) {
	ctx := context.Background()
	store := setupTest(t)
	err := store.Set(ctx, "key1", "value1", 5*time.Millisecond)
	assert.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	val, err := store.Get(ctx, "key1")
	fmt.Println(val)
	assert.Error(t, err)
}

func TestRedisDelete(t *testing.T) {
	ctx := context.Background()
	store := setupTest(t)
	err := store.Set(ctx, "key1", "value1", 90*time.Second)
	assert.NoError(t, err)

	err = store.Delete(ctx, "key1")
	assert.NoError(t, err)

	_, err = store.Get(ctx, "key1")
	assert.Error(t, err)
}

func TestRedisClear(t *testing.T) {
	ctx := context.Background()
	store := setupTest(t)
	err := store.Set(ctx, "key1", "value1", 60*time.Second)
	assert.NoError(t, err)
	err = store.Set(ctx, "key2", "value2", 60*time.Second)
	assert.NoError(t, err)

	err = store.Clear(ctx)
	assert.NoError(t, err)

	_, err1 := store.Get(ctx, "key1")
	_, err2 := store.Get(ctx, "key2")
	assert.Error(t, err1)
	assert.Error(t, err2)
}

func TestRedisSetGetWithJSONMarshalling(t *testing.T) {
	ctx := context.Background()
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: true,
		UseIntegerForTTL:   false,
	})
	err := store.Clear(ctx)
	assert.NoError(t, err)

	type TestPerson struct {
		Name    string   `json:"name"`
		Age     int      `json:"age"`
		Hobbies []string `json:"hobbies"`
	}
	// Test with a struct
	testData := TestPerson{
		Name: "John Doe",
		Age:  30,
		Hobbies: []string{
			"reading",
			"swimming",
		},
	}

	err = store.Set(ctx, "complex_key", testData, 60*time.Second)
	assert.NoError(t, err)

	value, err := store.Get(ctx, "complex_key")
	assert.NoError(t, err)

	// Convert the returned value to TestPerson
	var retrievedData TestPerson
	retrievedDataMap, ok := value.(map[string]interface{})
	assert.True(t, ok, "Retrieved value should be a map")

	jsonData, err := json.Marshal(retrievedDataMap)
	assert.NoError(t, err)

	err = json.Unmarshal(jsonData, &retrievedData)
	assert.NoError(t, err)

	// Use deep.Equal for comparison
	if diff := deep.Equal(testData, retrievedData); diff != nil {
		t.Error(diff)
	}
}

func TestRedisSetGetWithJSONMarshallingAndIntegerTTL(t *testing.T) {
	ctx := context.Background()
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: true,
		UseIntegerForTTL:   true,
	})
	err := store.Clear(ctx)
	assert.NoError(t, err)

	err = store.Set(ctx, "key1", "value1", 60*time.Second)
	assert.NoError(t, err)

	value, err := store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", value)
}

func TestRedisGetExpiredWithJSONMarshalling(t *testing.T) {
	ctx := context.Background()
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: true,
		UseIntegerForTTL:   false,
	})
	err := store.Clear(ctx)
	assert.NoError(t, err)

	err = store.Set(ctx, "key1", "value1", 10*time.Millisecond)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	_, err = store.Get(ctx, "key1")
	assert.Error(t, err)
}

func TestRedisGetExpiredWithJSONMarshallingAndIntegerTTL(t *testing.T) {
	ctx := context.Background()
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: true,
		UseIntegerForTTL:   true,
	})
	err := store.Clear(ctx)
	assert.NoError(t, err)

	// we can use very short millisecond times for ttl because we are using integer for ttl
	err = store.Set(ctx, "key1", "value1", 2*time.Millisecond)
	assert.NoError(t, err)

	time.Sleep(5 * time.Millisecond)

	_, err = store.Get(ctx, "key1")
	assert.Error(t, err)
}

func TestRedisSetGetWithoutJSONMarshalling(t *testing.T) {
	ctx := context.Background()
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: false,
		UseIntegerForTTL:   false,
	})
	err := store.Clear(ctx)
	assert.NoError(t, err)

	err = store.Set(ctx, "key1", "value1", 60*time.Second)
	assert.NoError(t, err)

	value, err := store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", value)
}

// func TestRedisClose(t *testing.T) {
// 	store := setupTest(t)
// 	err := store.Close()
// 	assert.NoError(t, err)
// }
