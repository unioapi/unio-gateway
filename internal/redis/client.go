package redis

import (
	"context"
	"fmt"

	"github.com/ThankCat/unio-api/internal/config"
	"github.com/redis/go-redis/v9"
)

// OpenRedis 创建 Redis client，并在启动期做一次 ping。
func OpenRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	// TODO(阶段2/production): [GAP-2-007] 将 Redis timeout、pool size 和 retry 策略从 config 传入，避免生产环境使用默认连接参数。
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
