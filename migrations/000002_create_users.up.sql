-- 用户账号主体，承载登录身份和用户归属边界。
CREATE TABLE users (
    -- id: 主键。--
    id BIGSERIAL PRIMARY KEY,

    -- email: 用户登录邮箱。--
    email TEXT NOT NULL,

    -- password_hash: 用户密码哈希。--
    password_hash TEXT NOT NULL,

    -- display_name: 用户展示名称。--
    display_name TEXT NOT NULL,

    -- created_at: 记录创建时间。--
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- updated_at: 记录更新时间。--
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- TODO(阶段2/production): [GAP-2-008] 统一 updated_at 更新策略，后续在 trigger 和显式 SQL 更新之间选定一种，避免不同表行为不一致。

-- 用户邮箱登录需要大小写不敏感唯一。
CREATE UNIQUE INDEX idx_users_email_lower ON users (lower(email));
