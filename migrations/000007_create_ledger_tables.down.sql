-- ledger_entries 依赖 user_balances 的 user 余额语义，回滚时先删除账本流水。
DROP TABLE IF EXISTS ledger_entries;
ALTER TABLE request_records
    DROP CONSTRAINT IF EXISTS uq_request_records_id_user;
DROP TABLE IF EXISTS user_balances;
