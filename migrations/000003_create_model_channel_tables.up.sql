-- providers 表示业务服务商，例如 OpenAI、Anthropic；它不是 Go adapter 接口。
CREATE TABLE providers (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    adapter TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('enabled', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- channels 表示某个 provider 下的一条具体上游渠道。
CREATE TABLE channels (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES providers (id),
    name TEXT NOT NULL,
    base_url TEXT NOT NULL,
    credential_ref TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('enabled', 'disabled')),
    priority INTEGER NOT NULL CHECK (priority >= 0),
    timeout_ms INTEGER CHECK (
      timeout_ms IS NULL OR timeout_ms > 0
    ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider_id, name)
);

-- models 表示 Unio 对外暴露的模型。
CREATE TABLE models (
    id BIGSERIAL PRIMARY KEY,
    model_id TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    owned_by TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('enabled', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- channel_models 表示某条 channel 能服务哪个模型，以及转发给上游时使用的模型名。
CREATE TABLE channel_models (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels (id),
    model_id BIGINT NOT NULL REFERENCES models (id),
    upstream_model TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('enabled', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (channel_id, model_id)
);

-- routing 查询会按 provider、model 和 channel 关系做 join，这些索引用于稳定查询成本。
CREATE INDEX idx_channels_provider_id ON channels(provider_id);
CREATE INDEX idx_channel_models_channel_id ON channel_models(channel_id);
CREATE INDEX idx_channel_models_model_id ON channel_models(model_id);
CREATE INDEX idx_channels_priority ON channels(priority, id);