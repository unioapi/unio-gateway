package ratelimit

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore 使用 Redis 保存固定窗口限流计数。
type RedisStore struct {
	client       redis.Cmdable
	keyNamespace string
	now          func() time.Time
}

// NewRedisStore 创建 Redis 限流存储。
func NewRedisStore(client redis.Cmdable, keyNamespace string) *RedisStore {
	return &RedisStore{
		client:       client,
		keyNamespace: keyNamespace,
		now:          time.Now,
	}
}

// Increment 增加 subject 对应窗口内的请求计数，并返回当前计数和重置时间。
func (s *RedisStore) Increment(ctx context.Context, key string, window time.Duration) (CountResult, error) {
	redisKey := redisKeyForSubject(s.keyNamespace, key)

	var countCmd *redis.IntCmd
	var ttlCmd *redis.DurationCmd

	_, err := s.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		// 在 Redis MULTI/EXEC 中完成计数和 TTL 设置，避免 INCR 成功但 EXPIRE 失败留下永久 key。
		countCmd = pipe.Incr(ctx, redisKey)

		// ExpireNX 只在 key 没有过期时间时设置 TTL，保证固定窗口不会被每次请求刷新。
		pipe.ExpireNX(ctx, redisKey, window)

		ttlCmd = pipe.TTL(ctx, redisKey)

		return nil
	})
	if err != nil {
		return CountResult{}, err
	}

	return CountResult{
		Count:   countCmd.Val(),
		ResetAt: s.now().Add(ttlCmd.Val()),
	}, nil
}

func redisKeyForSubject(namespace string, subject string) string {
	namespace = strings.Trim(namespace, ":")
	if namespace == "" {
		namespace = "unio"
	}
	return namespace + ":ratelimit:" + subject
}
