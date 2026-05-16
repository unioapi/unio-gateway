-- prices 表示模型售卖价配置，属于后台可管理的业务数据。
CREATE TABLE prices (
    id BIGSERIAL PRIMARY KEY,
    model_id BIGINT NOT NULL REFERENCES models(id),
    currency TEXT NOT NULL CHECK (currency <> ''),
    pricing_unit TEXT NOT NULL CHECK ( pricing_unit = 'per_1m_tokens' ),
    input_price NUMERIC(20, 10) NOT NULL CHECK ( input_price >= 0 ),
    output_price NUMERIC(20, 10) NOT NULL CHECK ( output_price >= 0 ),
    cached_input_price NUMERIC(20, 10) CHECK (
        cached_input_price IS NULL OR cached_input_price >= 0
    ),
    reasoning_output_price NUMERIC(20, 10) CHECK (
        reasoning_output_price IS NULL OR reasoning_output_price >= 0
    ),
    status TEXT NOT NULL CHECK ( status IN ('enabled', 'disabled') ),
    effective_from TIMESTAMPTZ NOT NULL,
    effective_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (effective_to IS NULL OR effective_to > effective_from)
);

-- price_snapshots 表示一次请求结算时使用的价格副本。
CREATE TABLE price_snapshots (
    id BIGSERIAL PRIMARY KEY,
    request_record_id BIGINT NOT NULL UNIQUE REFERENCES request_records(id),
    price_id BIGINT REFERENCES prices(id),
    currency TEXT NOT NULL CHECK (currency <> ''),
    pricing_unit TEXT NOT NULL CHECK (pricing_unit = 'per_1m_tokens'),
    input_price NUMERIC(20, 10) NOT NULL CHECK (input_price >= 0),
    output_price NUMERIC(20, 10) NOT NULL CHECK (output_price >= 0),
    cached_input_price NUMERIC(20, 10) CHECK (
        cached_input_price IS NULL OR cached_input_price >= 0
    ),
    reasoning_output_price NUMERIC(20, 10) CHECK (
        reasoning_output_price IS NULL OR reasoning_output_price >= 0
    ),
    formula_version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- active price 查询会按 model、status 和生效时间窗口过滤。
CREATE INDEX idx_prices_model_status_effective
    ON prices(model_id, status, effective_from DESC, id DESC);

