package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenPostgres 创建 PostgreSQL 连接池，并在启动期做一次 ping。
func OpenPostgres(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	// TODO(阶段2/production): 从 config 注入 pgxpool 参数，包括 max conns、min conns、max conn lifetime、idle time 和健康检查策略。
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	// TODO(阶段2/production): 启动期校验 migration 版本，避免服务在 schema 未迁移或版本不匹配时继续启动。

	return pool, nil
}
