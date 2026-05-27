-- Ledger reservation 是一次请求的余额预授权事实。
CREATE TABLE ledger_reservations (
    -- id: 主键。--
    id BIGSERIAL PRIMARY KEY,

    -- user_id: 预授权所属用户 ID。--
    user_id BIGINT NOT NULL REFERENCES users (id),

    -- request_record_id: 预授权对应的请求记录 ID。--
    request_record_id BIGINT NOT NULL,

    -- currency: 预授权币种。--
    currency TEXT NOT NULL CHECK (currency <> ''),

    -- status: 预授权状态。--
    status TEXT NOT NULL CHECK (status IN ('authorized', 'captured', 'released')),

    -- authorized_amount: 实际冻结金额。--
    authorized_amount NUMERIC(20, 10) NOT NULL CHECK (authorized_amount > 0),

    -- captured_amount: 已确认扣费金额。--
    captured_amount NUMERIC(20, 10) NOT NULL DEFAULT 0 CHECK (captured_amount >= 0),

    -- released_amount: 已释放冻结金额。--
    released_amount NUMERIC(20, 10) NOT NULL DEFAULT 0 CHECK (released_amount >= 0),

    -- estimated_amount: 本次请求的风险估算金额。--
    estimated_amount NUMERIC(20, 10) NOT NULL CHECK (estimated_amount > 0),

    -- capture_ledger_entry_id: capture 成功后对应的扣费流水 ID。--
    capture_ledger_entry_id BIGINT UNIQUE,

    -- idempotency_key: 预授权幂等键。--
    idempotency_key TEXT NOT NULL UNIQUE CHECK (idempotency_key <> ''),

    -- reason: 预授权业务原因。--
    reason TEXT NOT NULL CHECK (reason <> ''),

    -- created_at: 记录创建时间。--
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- updated_at: 记录更新时间。--
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- captured_at: capture 完成时间。--
    captured_at TIMESTAMPTZ,

    -- released_at: release 完成时间。--
    released_at TIMESTAMPTZ,

    -- 一个请求只能有一笔预授权。--
    UNIQUE (request_record_id),

    -- billing exception 需要用组合外键精确指向同用户同请求预授权。--
    CONSTRAINT uq_ledger_reservations_id_user_request
        UNIQUE (id, user_id, request_record_id),

    -- 实际冻结金额不能超过估算金额。--
    CONSTRAINT ck_ledger_reservations_authorized_not_above_estimated CHECK (
        authorized_amount <= estimated_amount
    ),

    -- 预授权必须保证 request_record 和 user 归属一致。--
    CONSTRAINT fk_ledger_reservations_request_user
        FOREIGN KEY (request_record_id, user_id)
            REFERENCES request_records (id, user_id),

    -- capture 和 release 的金额合计不能超过实际冻结金额。--
    CONSTRAINT ck_ledger_reservations_amount_sum CHECK (
        captured_amount + released_amount <= authorized_amount
    ),

    -- 不同预授权状态必须匹配对应金额、流水和时间字段。--
    CONSTRAINT ck_ledger_reservations_status_amounts CHECK (
        (
            status = 'authorized'
            AND captured_amount = 0
            AND released_amount = 0
            AND capture_ledger_entry_id IS NULL
            AND captured_at IS NULL
            AND released_at IS NULL
        )
        OR
        (
            status = 'captured'
            AND captured_amount > 0
            AND captured_amount + released_amount = authorized_amount
            AND capture_ledger_entry_id IS NOT NULL
            AND captured_at IS NOT NULL
            AND (
                (
                    released_amount = 0
                    AND released_at IS NULL
                )
                OR
                (
                    released_amount > 0
                    AND released_at IS NOT NULL
                )
            )
        )
        OR
        (
            status = 'released'
            AND captured_amount = 0
            AND released_amount = authorized_amount
            AND capture_ledger_entry_id IS NULL
            AND captured_at IS NULL
            AND released_at IS NOT NULL
        )
    ),

    -- capture 流水必须属于同一用户和同一请求。--
    CONSTRAINT fk_ledger_reservations_capture_entry
        FOREIGN KEY (capture_ledger_entry_id, user_id, request_record_id)
            REFERENCES ledger_entries (id, user_id, request_record_id)
);

-- 后台账单和审计常按用户倒序查看预授权。
CREATE INDEX idx_ledger_reservations_user_created_at
    ON ledger_reservations (user_id, created_at DESC, id DESC);

-- worker recovery 会扫描仍处于 authorized 状态的旧预授权。
CREATE INDEX idx_ledger_reservations_authorized_created_at
    ON ledger_reservations (created_at, id)
    WHERE status = 'authorized';
