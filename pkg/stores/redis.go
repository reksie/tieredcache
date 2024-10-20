package stores

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/reksie/memocache/pkg/interfaces"
)

type redisStore struct {
	name   string
	client *redis.Client
}

type redisItem struct {
	Value     any       `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
}

func CreateRedisStore(name string, client *redis.Client) interfaces.CacheStore {
	return &redisStore{
		name:   name,
		client: client,
	}
}

func (r *redisStore) Name() string {
	return r.name
}

func (r *redisStore) Set(key string, value any, ttl time.Duration) error {

	// for some reason redis is not using the ttl
	item := redisItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.Marshal(item)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return r.client.Set(ctx, key, data, ttl).Err() // Note: We're not using Redis TTL here
}

func (r *redisStore) Get(key string) (any, error) {
	ctx := context.Background()
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, errors.New("key not found in cache")
	} else if err != nil {
		return nil, err
	}

	var item redisItem
	err = json.Unmarshal(data, &item)
	if err != nil {
		return nil, err
	}

	if time.Now().After(item.ExpiresAt) {
		r.Delete(key) // Delete expired key
		return nil, errors.New("key expired")
	}

	return item.Value, nil
}

func (r *redisStore) Delete(key string) error {
	ctx := context.Background()
	return r.client.Del(ctx, key).Err()
}

func (r *redisStore) Clear() error {
	ctx := context.Background()
	return r.client.FlushDB(ctx).Err()
}

func (r *redisStore) Close() error {
	return r.client.Close()
}
