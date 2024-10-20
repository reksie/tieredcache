package stores

import (
	"context"
	"encoding/json"
	"fmt"
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
	ctx := context.Background()

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
		panic(err)
	}

	endpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		panic(err)
	}

	if localTest {
		endpoint = "localhost:6379"
	}

	fmt.Println(endpoint)

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
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: false,
		UseIntegerForTTL:   true,
	})
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

func TestRedisSetGetWithJSONMarshalling(t *testing.T) {
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: true,
		UseIntegerForTTL:   false,
	})
	err := store.Clear()
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

	err = store.Set("complex_key", testData, 60*time.Second)
	assert.NoError(t, err)

	value, err := store.Get("complex_key")
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
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: true,
		UseIntegerForTTL:   true,
	})
	err := store.Clear()
	assert.NoError(t, err)

	err = store.Set("key1", "value1", 60*time.Second)
	assert.NoError(t, err)

	value, err := store.Get("key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", value)
}

func TestRedisGetExpiredWithJSONMarshalling(t *testing.T) {
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: true,
		UseIntegerForTTL:   false,
	})
	err := store.Clear()
	assert.NoError(t, err)

	err = store.Set("key1", "value1", 10*time.Millisecond)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	_, err = store.Get("key1")
	assert.Error(t, err)
}

func TestRedisGetExpiredWithJSONMarshallingAndIntegerTTL(t *testing.T) {
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: true,
		UseIntegerForTTL:   true,
	})
	err := store.Clear()
	assert.NoError(t, err)

	err = store.Set("key1", "value1", 10*time.Millisecond)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	_, err = store.Get("key1")
	assert.Error(t, err)
}

func TestRedisSetGetWithoutJSONMarshalling(t *testing.T) {
	store := CreateRedisStore("test_store", redisClient, RedisStoreConfig{
		UseJSONMarshalling: false,
		UseIntegerForTTL:   false,
	})
	err := store.Clear()
	assert.NoError(t, err)

	err = store.Set("key1", "value1", 60*time.Second)
	assert.NoError(t, err)

	value, err := store.Get("key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", value)
}

func TestRedisClose(t *testing.T) {
	store := setupTest(t)
	err := store.Close()
	assert.NoError(t, err)
}
