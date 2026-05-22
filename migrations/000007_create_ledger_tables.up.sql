-- user_balances 表示用户当前余额投影；余额事实来源仍然是 ledger_entries。
CREATE TABLE user_balances (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id),
    currency TEXT NOT NULL CHECK (currency <> ''),
    balance NUMERIC(20, 10) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    reserved_balance NUMERIC(20, 10) NOT NULL DEFAULT 0 CHECK (reserved_balance >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, currency),
    CONSTRAINT ck_user_balances_reserved_not_above_balance CHECK (reserved_balance <= balance)
);

-- request_records.id 本身已经唯一；这里增加组合唯一约束，让 ledger_entries 可以用组合外键保证 request 和 user 归属一致。
ALTER TABLE request_records
    ADD CONSTRAINT uq_request_records_id_user UNIQUE (id, user_id);

-- ledger_entries 表示每一次用户余额变化的账本事实。
CREATE TABLE ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id),
    request_record_id BIGINT,
    entry_type TEXT NOT NULL CHECK (
        entry_type IN (
            'credit',
            'debit',
            'refund',
            'adjustment_credit',
            'adjustment_debit'
        )
    ),
    amount NUMERIC(20, 10) NOT NULL CHECK (amount > 0),
    currency TEXT NOT NULL CHECK (currency <> ''),
    balance_before NUMERIC(20, 10) NOT NULL CHECK (balance_before >= 0),
    balance_after NUMERIC(20, 10) NOT NULL CHECK (balance_after >= 0),
    idempotency_key TEXT NOT NULL UNIQUE CHECK (idempotency_key <> ''),
    reason TEXT NOT NULL CHECK (reason <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- 加款类流水必须让余额增加；扣款类流水必须让余额减少。
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

    CONSTRAINT fk_ledger_entries_request_user
        FOREIGN KEY (request_record_id, user_id)
            REFERENCES request_records (id, user_id),

    CONSTRAINT uq_ledger_entries_id_user_request
        UNIQUE (id, user_id, request_record_id)
);

-- 后台账单和审计常按用户倒序查看流水。
CREATE INDEX idx_ledger_entries_user_created_at ON ledger_entries (user_id, created_at DESC, id DESC);

-- 请求详情页需要从 request_record 找到对应扣费或退款流水。
CREATE INDEX idx_ledger_entries_request_record_id ON ledger_entries (request_record_id)
    WHERE
        request_record_id IS NOT NULL;
