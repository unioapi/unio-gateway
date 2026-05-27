-- Channel model 表示某条 channel 能服务哪个模型及对应上游模型名。
CREATE TABLE channel_models (
    -- id: 主键。--
    id BIGSERIAL PRIMARY KEY,

    -- channel_id: 可服务该模型的 channel ID。--
    channel_id BIGINT NOT NULL REFERENCES channels (id),

    -- model_id: Unio 侧模型 ID。--
    model_id BIGINT NOT NULL REFERENCES models (id),

    -- upstream_model: 转发到上游时使用的模型名。--
    upstream_model TEXT NOT NULL,

    -- status: channel-model 映射启停状态。--
    status TEXT NOT NULL CHECK (status IN ('enabled', 'disabled')),

    -- created_at: 记录创建时间。--
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- updated_at: 记录更新时间。--
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- 同一 channel 对同一模型只能配置一次。--
    UNIQUE (channel_id, model_id)
);

-- routing 会按 channel 查找可服务模型。
CREATE INDEX idx_channel_models_channel_id ON channel_models (channel_id);

-- /v1/models 和 routing 会按模型反查可用 channel。
CREATE INDEX idx_channel_models_model_id ON channel_models (model_id);
