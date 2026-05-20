package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThankCat/unio-api/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenPostgres 创建 PostgreSQL 连接池，并在启动期做一次 ping。
func OpenPostgres(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	if cfg.URL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	// TODO(阶段2/production): [GAP-2-006] 启动期校验 migration 版本，避免服务在 schema 未迁移或版本不匹配时继续启动。

	return pool, nil
}
