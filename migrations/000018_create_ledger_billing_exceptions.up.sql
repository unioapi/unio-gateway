-- Ledger billing exception 是结算中的平台核销或风险敞口审计事实。
CREATE TABLE ledger_billing_exceptions (
    -- id: 主键。--
    id BIGSERIAL PRIMARY KEY,

    -- user_id: 异常所属用户 ID。--
    user_id BIGINT NOT NULL REFERENCES users (id),

    -- request_record_id: 异常对应的请求记录 ID。--
    request_record_id BIGINT NOT NULL,

    -- reservation_id: 异常对应的预授权 ID。--
    reservation_id BIGINT NOT NULL,

    -- event_type: 异常事件类型。--
    event_type TEXT NOT NULL CHECK (event_type IN ('write_off', 'risk_exposure')),

    -- actual_amount: 真实应结算金额，风险敞口场景为空。--
    actual_amount NUMERIC(20, 10) CHECK (actual_amount IS NULL OR actual_amount > 0),

    -- captured_amount: 已从用户冻结金额中 capture 的金额。--
    captured_amount NUMERIC(20, 10) NOT NULL CHECK (captured_amount >= 0),

    -- platform_amount: 平台承担的金额。--
    platform_amount NUMERIC(20, 10) NOT NULL CHECK (platform_amount > 0),

    -- currency: 异常金额币种。--
    currency TEXT NOT NULL CHECK (currency <> ''),

    -- reason_code: 稳定原因码。--
    reason_code TEXT NOT NULL CHECK (reason_code <> ''),

    -- reason: 可审计的业务原因。--
    reason TEXT NOT NULL CHECK (reason <> ''),

    -- created_at: 记录创建时间。--
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- 一个请求最多写入一条 billing exception。--
    UNIQUE (request_record_id),

    -- 一笔预授权最多写入一条 billing exception。--
    UNIQUE (reservation_id),

    -- 异常必须保证 request_record 和 user 归属一致。--
    CONSTRAINT fk_ledger_billing_exceptions_request_user
        FOREIGN KEY (request_record_id, user_id)
            REFERENCES request_records (id, user_id),

    -- 异常必须指向同用户同请求的预授权。--
    CONSTRAINT fk_ledger_billing_exceptions_reservation
        FOREIGN KEY (reservation_id, user_id, request_record_id)
            REFERENCES ledger_reservations (id, user_id, request_record_id),

    -- write_off 记录平台核销差额，risk_exposure 记录无可靠 usage 的风险敞口。--
    CONSTRAINT ck_ledger_billing_exceptions_amounts CHECK (
        (
            event_type = 'write_off'
            AND actual_amount IS NOT NULL
            AND captured_amount < actual_amount
            AND platform_amount = actual_amount - captured_amount
        )
        OR
        (
            event_type = 'risk_exposure'
            AND actual_amount IS NULL
            AND captured_amount = 0
        )
    )
);

-- 后台账单和审计常按用户倒序查看 billing exception。
CREATE INDEX idx_ledger_billing_exceptions_user_created_at
    ON ledger_billing_exceptions (user_id, created_at DESC, id DESC);
