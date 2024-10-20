package stores

import (
	"context"
	"testing"
	"time"

	"github.com/allegro/bigcache/v3"
)

func createTestCache() (*bigcache.BigCache, error) {
	config := bigcache.DefaultConfig(10 * time.Minute)
	return bigcache.New(context.Background(), config)
}

func TestCreateMemoryStore(t *testing.T) {
	cache, err := createTestCache()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	store := CreateMemoryStore("test_store", cache)
	if store.Name() != "test_store" {
		t.Errorf("Expected store name to be 'test_store', got '%s'", store.Name())
	}
}

func TestSet(t *testing.T) {
	ctx := context.Background()
	cache, err := createTestCache()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	store := CreateMemoryStore("test_store", cache)
	err = store.Set(ctx, "key1", "value1", 60*time.Second)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	cache, err := createTestCache()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	store := CreateMemoryStore("test_store", cache)
	err = store.Set(ctx, "key1", "value1", 60*time.Second)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	value, err := store.Get(ctx, "key1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", value)
	}

	_, err = store.Get(ctx, "non_existent_key")
	if err == nil {
		t.Error("Expected error for non-existent key, got nil")
	}
}

func TestGetExpired(t *testing.T) {
	ctx := context.Background()
	cache, err := createTestCache()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	store := CreateMemoryStore("test_store", cache)
	err = store.Set(ctx, "key1", "value1", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = store.Get(ctx, "key1")
	if err == nil {
		t.Error("Expected error for expired key, got nil")
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	cache, err := createTestCache()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	store := CreateMemoryStore("test_store", cache)
	err = store.Set(ctx, "key1", "value1", 60*time.Second)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = store.Delete(ctx, "key1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	_, err = store.Get(ctx, "key1")
	if err == nil {
		t.Error("Expected error after deletion, got nil")
	}
}

func TestClear(t *testing.T) {
	ctx := context.Background()
	cache, err := createTestCache()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	store := CreateMemoryStore("test_store", cache)
	err = store.Set(ctx, "key1", "value1", 60*time.Second)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	err = store.Set(ctx, "key2", "value2", 60*time.Second)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = store.Clear(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	_, err1 := store.Get(ctx, "key1")
	_, err2 := store.Get(ctx, "key2")
	if err1 == nil || err2 == nil {
		t.Error("Expected errors after clearing, got nil")
	}
}

func TestClose(t *testing.T) {
	cache, err := createTestCache()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	store := CreateMemoryStore("test_store", cache)
	err = store.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
