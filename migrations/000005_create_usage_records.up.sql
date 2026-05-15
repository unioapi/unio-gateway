-- usage_records 表示一次请求最终用于计费和审计的 token 用量。
CREATE TABLE usage_records (
    id BIGSERIAL PRIMARY KEY,
    request_record_id BIGINT NOT NULL UNIQUE REFERENCES request_records(id),
    prompt_tokens BIGINT NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0),
    completion_tokens BIGINT NOT NULL DEFAULT 0 CHECK (completion_tokens >= 0),
    total_tokens BIGINT NOT NULL DEFAULT 0 CHECK (total_tokens >= 0),
    cached_tokens BIGINT NOT NULL DEFAULT 0 CHECK (cached_tokens >= 0),
    reasoning_tokens BIGINT NOT NULL DEFAULT 0 CHECK (reasoning_tokens >= 0),
    source TEXT NOT NULL CHECK (source IN ('upstream_response', 'upstream_stream')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (total_tokens = prompt_tokens + completion_tokens),
    CHECK (cached_tokens <= prompt_tokens),
    CHECK (reasoning_tokens <= completion_tokens)
);