-- Usage record 是一次请求最终用于计费和审计的 token 用量事实。
CREATE TABLE usage_records (
    -- id: 主键。--
    id BIGSERIAL PRIMARY KEY,

    -- request_record_id: 对应的请求记录 ID。--
    request_record_id BIGINT NOT NULL UNIQUE REFERENCES request_records (id),

    -- prompt_tokens: 输入 token 数。--
    prompt_tokens BIGINT NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0),

    -- completion_tokens: 输出 token 数。--
    completion_tokens BIGINT NOT NULL DEFAULT 0 CHECK (completion_tokens >= 0),

    -- total_tokens: 总 token 数。--
    total_tokens BIGINT NOT NULL DEFAULT 0 CHECK (total_tokens >= 0),

    -- cached_tokens: 命中缓存的输入 token 数。--
    cached_tokens BIGINT NOT NULL DEFAULT 0 CHECK (cached_tokens >= 0),

    -- reasoning_tokens: reasoning 输出 token 数。--
    reasoning_tokens BIGINT NOT NULL DEFAULT 0 CHECK (reasoning_tokens >= 0),

    -- source: usage 来源。--
    source TEXT NOT NULL CHECK (source IN ('upstream_response', 'upstream_stream')),

    -- created_at: 记录创建时间。--
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- total_tokens 必须等于输入和输出 token 之和。--
    CHECK (total_tokens = prompt_tokens + completion_tokens),

    -- cached_tokens 不能超过输入 token。--
    CHECK (cached_tokens <= prompt_tokens),

    -- reasoning_tokens 不能超过输出 token。--
    CHECK (reasoning_tokens <= completion_tokens)
);
