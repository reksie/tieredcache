package stores

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/reksie/tieredcache/pkg/interfaces"
)

type RedisStoreConfig struct {
	UseJSONMarshalling bool
	UseIntegerForTTL   bool
}

type redisStore struct {
	name   string
	client *redis.Client
	config RedisStoreConfig
}

type redisItem struct {
	Value     any         `json:"value"`
	ExpiresAt interface{} `json:"expires_at"`
}

func CreateRedisStore(name string, client *redis.Client, config RedisStoreConfig) interfaces.CacheStore {
	return &redisStore{
		name:   name,
		client: client,
		config: config,
	}
}

func (r *redisStore) Name() string {
	return r.name
}

func (r *redisStore) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if r.config.UseJSONMarshalling {
		expiresAt := time.Now().Add(ttl)
		item := redisItem{
			Value: value,
		}

		if r.config.UseIntegerForTTL {
			item.ExpiresAt = expiresAt.Unix()
		} else {
			item.ExpiresAt = expiresAt
		}

		data, err := json.Marshal(item)
		if err != nil {
			return err
		}

		return r.client.Set(ctx, key, data, ttl).Err()
	}

	// If not using JSON marshalling, use Redis's built-in TTL
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *redisStore) Get(ctx context.Context, key string) (any, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, errors.New("key not found in cache")
	} else if err != nil {
		return nil, err
	}

	if !r.config.UseJSONMarshalling {
		// If not using JSON marshalling, return the raw data
		return string(data), nil
	}

	var item redisItem
	err = json.Unmarshal(data, &item)
	if err != nil {
		return nil, err
	}

	var expiresAt time.Time
	if r.config.UseIntegerForTTL {
		unixTime, ok := item.ExpiresAt.(float64)
		if !ok {
			return nil, errors.New("invalid expiration time format")
		}
		expiresAt = time.Unix(int64(unixTime), 0)
	} else {
		expiresAtStr, ok := item.ExpiresAt.(string)
		if !ok {
			return nil, errors.New("invalid expiration time format")
		}
		expiresAt, err = time.Parse(time.RFC3339, expiresAtStr)
		if err != nil {
			return nil, err
		}
	}

	if time.Now().After(expiresAt) {
		r.Delete(ctx, key) // Delete expired key
		return nil, errors.New("key expired")
	}

	return item.Value, nil

}

func (r *redisStore) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *redisStore) Clear(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

func (r *redisStore) Close() error {
	return r.client.Close()
}
