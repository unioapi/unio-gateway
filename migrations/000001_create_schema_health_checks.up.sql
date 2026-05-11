-- 用于验证 migration 流程已经跑通，不承载业务含义。
-- TODO(阶段2/production): 引入正式 migration runner 和 schema 版本检查后，决定保留该开发期验证表还是迁移到专门的 schema_migrations 健康检查。
CREATE TABLE schema_health_checks (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
