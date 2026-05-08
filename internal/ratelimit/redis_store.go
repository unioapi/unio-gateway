package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore 使用 Redis 保存固定窗口限流计数。
type RedisStore struct {
	client redis.Cmdable
	now    func() time.Time
}

// NewRedisStore 创建 Redis 限流存储。
func NewRedisStore(client redis.Cmdable) *RedisStore {
	return &RedisStore{
		client: client,
		now:    time.Now,
	}
}

// Increment 增加 subject 对应窗口内的请求计数，并返回当前计数和重置时间。
func (s *RedisStore) Increment(ctx context.Context, key string, window time.Duration) (CountResult, error) {
	redisKey := redisKeyForSubject(key)

	count, err := s.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return CountResult{}, err
	}

	if count == 1 {
		if err := s.client.Expire(ctx, redisKey, window).Err(); err != nil {
			return CountResult{}, err
		}
	}

	ttl, err := s.client.TTL(ctx, redisKey).Result()
	if err != nil {
		return CountResult{}, err
	}

	return CountResult{
		Count:   count,
		ResetAt: s.now().Add(ttl),
	}, nil
}

func redisKeyForSubject(subject string) string {
	return "unio:ratelimit:" + subject
}
