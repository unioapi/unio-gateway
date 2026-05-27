-- Model 是 Unio 对外暴露和计费的模型目录。
CREATE TABLE models (
    -- id: 主键。--
    id BIGSERIAL PRIMARY KEY,

    -- model_id: OpenAI-compatible API 对外暴露的模型 ID。--
    model_id TEXT NOT NULL UNIQUE,

    -- display_name: 模型展示名称。--
    display_name TEXT NOT NULL,

    -- owned_by: 模型归属方展示字段。--
    owned_by TEXT NOT NULL,

    -- status: 模型启停状态。--
    status TEXT NOT NULL CHECK (status IN ('enabled', 'disabled')),

    -- created_at: 记录创建时间。--
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- updated_at: 记录更新时间。--
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
