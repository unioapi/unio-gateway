-- User balance 是用户当前余额投影，最终事实仍以 ledger_entries 为准。
CREATE TABLE user_balances (
    -- id: 主键。--
    id BIGSERIAL PRIMARY KEY,

    -- user_id: 余额所属用户 ID。--
    user_id BIGINT NOT NULL REFERENCES users (id),

    -- currency: 余额币种。--
    currency TEXT NOT NULL CHECK (currency <> ''),

    -- balance: 用户总余额。--
    balance NUMERIC(20, 10) NOT NULL DEFAULT 0 CHECK (balance >= 0),

    -- reserved_balance: 已冻结但尚未 capture/release 的余额。--
    reserved_balance NUMERIC(20, 10) NOT NULL DEFAULT 0 CHECK (reserved_balance >= 0),

    -- created_at: 记录创建时间。--
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- updated_at: 记录更新时间。--
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- 同一用户同一币种只能有一条余额投影。--
    UNIQUE (user_id, currency),

    -- 冻结余额不能超过总余额。--
    CONSTRAINT ck_user_balances_reserved_not_above_balance CHECK (reserved_balance <= balance)
);
