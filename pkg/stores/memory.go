package stores

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/reksie/memocache/pkg/interfaces"
)

type bigCacheStore struct {
	name  string
	cache *bigcache.BigCache
}

type cacheItem struct {
	Value     any       `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
}

func CreateMemoryStore(name string, cache *bigcache.BigCache) interfaces.CacheStore {
	return &bigCacheStore{
		name:  name,
		cache: cache,
	}
}

func (b *bigCacheStore) Name() string {
	return b.name
}

func (b *bigCacheStore) Set(key string, value any, ttl time.Duration) error {
	item := cacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return b.cache.Set(key, data)
}

func (b *bigCacheStore) Get(key string) (any, error) {
	data, err := b.cache.Get(key)
	if err != nil {
		if err == bigcache.ErrEntryNotFound {
			return nil, errors.New("key not found in cache")
		}
		return nil, err
	}

	var item cacheItem
	err = json.Unmarshal(data, &item)
	if err != nil {
		return nil, err
	}

	if time.Now().After(item.ExpiresAt) {
		b.cache.Delete(key)
		return nil, errors.New("key expired")
	}

	return item.Value, nil
}

func (b *bigCacheStore) Delete(key string) error {
	return b.cache.Delete(key)
}

func (b *bigCacheStore) Clear() error {
	return b.cache.Reset()
}

func (b *bigCacheStore) Close() error {
	return b.cache.Close()
}
