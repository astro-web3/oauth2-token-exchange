package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CachedToken struct {
	AccessToken string   `json:"access_token"`
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Groups      []string `json:"groups"`
}

type TokenCache interface {
	Get(ctx context.Context, patHash string) (*CachedToken, error)
	Set(ctx context.Context, patHash string, value *CachedToken, ttl time.Duration) error
}

type redisCache struct {
	client *redis.Client
}

func NewRedisClient(url string, poolSize int) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	opt.PoolSize = poolSize

	client := redis.NewClient(opt)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}

func NewTokenCache(client *redis.Client) TokenCache {
	return &redisCache{client: client}
}

func (r *redisCache) Get(ctx context.Context, patHash string) (*CachedToken, error) {
	key := fmt.Sprintf("authz:pat:%s", patHash)
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from redis: %w", err)
	}

	var token CachedToken
	if err := json.Unmarshal([]byte(val), &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached token: %w", err)
	}

	return &token, nil
}

func (r *redisCache) Set(ctx context.Context, patHash string, value *CachedToken, ttl time.Duration) error {
	key := fmt.Sprintf("authz:pat:%s", patHash)
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cached token: %w", err)
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set redis cache: %w", err)
	}

	return nil
}
