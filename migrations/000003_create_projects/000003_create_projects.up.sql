-- 项目是用户 API Key、模型策略和调用归属的业务容器。
CREATE TABLE projects (
    -- id: 主键。--
    id BIGSERIAL PRIMARY KEY,

    -- user_id: 项目所属用户 ID。--
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    -- name: 用户侧项目名称。--
    name TEXT NOT NULL,

    -- created_at: 记录创建时间。--
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- updated_at: 记录更新时间。--
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- 同一用户下项目名称不能重复。--
    UNIQUE (user_id, name)
);

-- 用户后台常按 user_id 查询项目列表。
CREATE INDEX idx_projects_user_id ON projects (user_id);
