-- prices 表示模型售卖价配置，属于后台可管理的业务数据。
-- TODO(阶段7/production): [GAP-7-009] prices 当前只表达客户侧售卖价，缺少 provider/channel 成本价快照会导致毛利、成本审计和 fallback 成本分析不完整；进入成本报表或多 channel 商业化前；增加 provider/channel cost price 与请求级 cost snapshot。
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

-- TODO(阶段7/production): [GAP-7-010] prices 允许同一 model/currency/pricing_unit 出现重叠 enabled 生效窗口，后台改价时可能导致结算价格不确定；开放价格后台管理前；用排他约束或事务化关停旧价格保证生效区间不重叠。

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
