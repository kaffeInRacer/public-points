package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"online-shop/pkg/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisClient struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisClient(redisURL string, logger *zap.Logger) *RedisClient {
	// Parse Redis URL
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Error("Failed to parse Redis URL, using defaults", zap.Error(err))
		opts = &redis.Options{
			Addr: "localhost:6379",
		}
	}

	client := redis.NewClient(opts)
	
	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("Failed to connect to Redis", zap.Error(err))
	} else {
		logger.Info("Connected to Redis successfully")
	}

	return &RedisClient{
		client: client,
		logger: logger,
	}
}

type Client struct {
	rdb *redis.Client
}

func NewClient(cfg *config.RedisConfig) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &Client{rdb: rdb}
}

// RedisClient methods
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	ctx := context.Background()
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return r.client.Set(ctx, key, data, expiration).Err()
}

func (r *RedisClient) Get(key string, dest ...interface{}) error {
	ctx := context.Background()
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found")
		}
		return fmt.Errorf("failed to get key: %w", err)
	}
	
	if len(dest) > 0 {
		return json.Unmarshal([]byte(val), dest[0])
	}
	return nil
}

func (r *RedisClient) GetString(key string) (string, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("key not found")
		}
		return "", fmt.Errorf("failed to get key: %w", err)
	}
	return val, nil
}

func (r *RedisClient) Delete(keys ...string) error {
	ctx := context.Background()
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisClient) Exists(key string) (bool, error) {
	ctx := context.Background()
	count, err := r.client.Exists(ctx, key).Result()
	return count > 0, err
}

func (r *RedisClient) SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	ctx := context.Background()
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}
	return r.client.SetNX(ctx, key, data, expiration).Result()
}

func (r *RedisClient) DeletePattern(pattern string) error {
	ctx := context.Background()
	
	// Use SCAN to find keys matching the pattern
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string
	
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}
	
	// Delete keys in batches
	if len(keys) > 0 {
		pipe := r.client.Pipeline()
		for _, key := range keys {
			pipe.Del(ctx, key)
		}
		_, err := pipe.Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
	}
	
	return nil
}

func (r *RedisClient) Ping() error {
	ctx := context.Background()
	return r.client.Ping(ctx).Err()
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Legacy Client methods for backward compatibility
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, data, expiration).Err()
}

func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.rdb.Exists(ctx, key).Result()
	return count > 0, err
}

func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return c.rdb.SetNX(ctx, key, data, expiration).Result()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

// Cache service for common operations
type CacheService struct {
	client *Client
}

func NewCacheService(client *Client) *CacheService {
	return &CacheService{client: client}
}

func (s *CacheService) CacheUser(ctx context.Context, userID string, user interface{}) error {
	key := fmt.Sprintf("user:%s", userID)
	return s.client.Set(ctx, key, user, 30*time.Minute)
}

func (s *CacheService) GetCachedUser(ctx context.Context, userID string, dest interface{}) error {
	key := fmt.Sprintf("user:%s", userID)
	return s.client.Get(ctx, key, dest)
}

func (s *CacheService) InvalidateUser(ctx context.Context, userID string) error {
	key := fmt.Sprintf("user:%s", userID)
	return s.client.Delete(ctx, key)
}

func (s *CacheService) CacheProduct(ctx context.Context, productID string, product interface{}) error {
	key := fmt.Sprintf("product:%s", productID)
	return s.client.Set(ctx, key, product, 1*time.Hour)
}

func (s *CacheService) GetCachedProduct(ctx context.Context, productID string, dest interface{}) error {
	key := fmt.Sprintf("product:%s", productID)
	return s.client.Get(ctx, key, dest)
}

func (s *CacheService) InvalidateProduct(ctx context.Context, productID string) error {
	key := fmt.Sprintf("product:%s", productID)
	return s.client.Delete(ctx, key)
}

func (s *CacheService) SetSession(ctx context.Context, sessionID string, data interface{}) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return s.client.Set(ctx, key, data, 24*time.Hour)
}

func (s *CacheService) GetSession(ctx context.Context, sessionID string, dest interface{}) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return s.client.Get(ctx, key, dest)
}

func (s *CacheService) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return s.client.Delete(ctx, key)
}