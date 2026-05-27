-- Ledger entry 是用户余额变化的账本事实。
CREATE TABLE ledger_entries (
    -- id: 主键。--
    id BIGSERIAL PRIMARY KEY,

    -- user_id: 流水所属用户 ID。--
    user_id BIGINT NOT NULL REFERENCES users (id),

    -- request_record_id: 触发该流水的请求记录 ID，非请求流水为空。--
    request_record_id BIGINT,

    -- entry_type: 流水类型。--
    entry_type TEXT NOT NULL CHECK (
        entry_type IN (
            'credit',
            'debit',
            'refund',
            'adjustment_credit',
            'adjustment_debit'
        )
    ),

    -- amount: 本次流水金额，必须为正数。--
    amount NUMERIC(20, 10) NOT NULL CHECK (amount > 0),

    -- currency: 流水币种。--
    currency TEXT NOT NULL CHECK (currency <> ''),

    -- balance_before: 流水发生前余额。--
    balance_before NUMERIC(20, 10) NOT NULL CHECK (balance_before >= 0),

    -- balance_after: 流水发生后余额。--
    balance_after NUMERIC(20, 10) NOT NULL CHECK (balance_after >= 0),

    -- idempotency_key: 流水幂等键。--
    idempotency_key TEXT NOT NULL UNIQUE CHECK (idempotency_key <> ''),

    -- reason: 流水业务原因。--
    reason TEXT NOT NULL CHECK (reason <> ''),

    -- created_at: 记录创建时间。--
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- 加款类流水必须让余额增加；扣款类流水必须让余额减少。--
    CONSTRAINT ck_ledger_entries_balance_math CHECK (
        (
            entry_type IN ('credit', 'refund', 'adjustment_credit')
            AND balance_after = balance_before + amount
        )
        OR
        (
            entry_type IN ('debit', 'adjustment_debit')
            AND balance_after = balance_before - amount
        )
    ),

    -- 请求关联流水必须保证 request_record 和 user 归属一致。--
    CONSTRAINT fk_ledger_entries_request_user
        FOREIGN KEY (request_record_id, user_id)
            REFERENCES request_records (id, user_id),

    -- reservation capture 需要用组合外键精确指向同用户同请求流水。--
    CONSTRAINT uq_ledger_entries_id_user_request
        UNIQUE (id, user_id, request_record_id)
);

-- 后台账单和审计常按用户倒序查看流水。
CREATE INDEX idx_ledger_entries_user_created_at ON ledger_entries (user_id, created_at DESC, id DESC);

-- 请求详情页需要从 request_record 找到对应扣费或退款流水。
CREATE INDEX idx_ledger_entries_request_record_id ON ledger_entries (request_record_id)
    WHERE request_record_id IS NOT NULL;
