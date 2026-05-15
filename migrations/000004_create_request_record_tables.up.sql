-- request_records 表示一次用户可见的 Unio API 请求。
CREATE TABLE request_records (
    id BIGSERIAL PRIMARY KEY,
    request_id TEXT NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    project_id BIGINT NOT NULL REFERENCES projects(id),
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id),
    requested_model_id TEXT NOT NULL,
    response_model_id TEXT,
    stream BOOLEAN NOT NULL,
    status TEXT NOT NULL CHECK ( status IN ('pending', 'running', 'succeeded', 'failed', 'canceled')),
    final_provider_id BIGINT REFERENCES providers(id),
    final_channel_id BIGINT REFERENCES channels(id),
    error_code TEXT,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- request_attempts 表示一次 logical request 下的一次上游 channel 尝试。
CREATE TABLE request_attempts (
    id BIGSERIAL PRIMARY KEY,
    request_record_id BIGINT NOT NULL REFERENCES request_records(id),
    attempt_index INTEGER NOT NULL CHECK (attempt_index >= 0),
    provider_id BIGINT NOT NULL REFERENCES providers(id),
    channel_id BIGINT NOT NULL REFERENCES channels(id),
    adapter_key TEXT NOT NULL,
    upstream_model TEXT NOT NULL,
    upstream_response_model TEXT,
    status TEXT NOT NULL CHECK ( status IN ('running', 'succeeded', 'failed', 'canceled')),
    upstream_status_code INTEGER CHECK (
        upstream_status_code IS NULL
        OR (upstream_status_code >= 100 AND upstream_status_code <= 599)
    ),
    upstream_request_id TEXT,
    error_code TEXT,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (request_record_id, attempt_index)
);

-- request_records.sql 常按用户、项目、API Key 和时间范围检索，用于账单、审计和后台请求日志。
CREATE INDEX idx_request_records_user_created_at ON request_records(user_id, created_at DESC);
CREATE INDEX idx_request_records_project_created_at ON request_records(project_id, created_at DESC);
CREATE INDEX idx_request_records_api_key_created_at ON request_records(api_key_id, created_at DESC);
CREATE INDEX idx_request_records_status_created_at ON request_records(status, created_at DESC);

-- request_attempts 常按 request_record_id 查询完整 fallback 过程。
CREATE INDEX idx_request_attempts_channel_created_at ON request_attempts(channel_id, created_at DESC);