-- name: CreateUserBalance :one
INSERT INTO
    user_balances (user_id, currency, balance)
VALUES
    (
        sqlc.arg (user_id),
        sqlc.arg (currency),
        sqlc.arg (balance)
    )
RETURNING
    id,
    user_id,
    currency,
    balance,
    created_at,
    updated_at;

-- name: GetUserBalance :one
SELECT
    id,
    user_id,
    currency,
    balance,
    created_at,
    updated_at
FROM
    user_balances
WHERE
    user_id = sqlc.arg (user_id)
  AND currency = sqlc.arg (currency);

-- name: GetUserBalanceForUpdate :one
SELECT
    id,
    user_id,
    currency,
    balance,
    created_at,
    updated_at
FROM
    user_balances
WHERE
    user_id = sqlc.arg (user_id)
  AND currency = sqlc.arg (currency)
    FOR UPDATE;

-- name: UpdateUserBalance :one
UPDATE user_balances
SET
    balance = sqlc.arg (balance),
    updated_at = now()
WHERE
    user_id = sqlc.arg (user_id)
  AND currency = sqlc.arg (currency)
RETURNING
    id,
    user_id,
    currency,
    balance,
    created_at,
    updated_at;

-- name: CreateLedgerEntry :one
INSERT INTO
    ledger_entries (
    user_id,
    request_record_id,
    entry_type,
    amount,
    currency,
    balance_before,
    balance_after,
    idempotency_key,
    reason
)
VALUES
    (
        sqlc.arg (user_id),
        sqlc.arg (request_record_id),
        sqlc.arg (entry_type),
        sqlc.arg (amount),
        sqlc.arg (currency),
        sqlc.arg (balance_before),
        sqlc.arg (balance_after),
        sqlc.arg (idempotency_key),
        sqlc.arg (reason)
    )
RETURNING
    id,
    user_id,
    request_record_id,
    entry_type,
    amount,
    currency,
    balance_before,
    balance_after,
    idempotency_key,
    reason,
    created_at;

-- name: GetLedgerEntryByIdempotencyKey :one
SELECT
    id,
    user_id,
    request_record_id,
    entry_type,
    amount,
    currency,
    balance_before,
    balance_after,
    idempotency_key,
    reason,
    created_at
FROM
    ledger_entries
WHERE
    idempotency_key = sqlc.arg (idempotency_key);

-- name: ListLedgerEntriesByUser :many
SELECT
    id,
    user_id,
    request_record_id,
    entry_type,
    amount,
    currency,
    balance_before,
    balance_after,
    idempotency_key,
    reason,
    created_at
FROM
    ledger_entries
WHERE
    user_id = sqlc.arg (user_id)
  AND currency = sqlc.arg (currency)
ORDER BY
    created_at DESC,
    id DESC
LIMIT
    sqlc.arg (limit_rows)
    OFFSET
    sqlc.arg (offset_rows);

-- name: ListLedgerEntriesByRequest :many
SELECT
    id,
    user_id,
    request_record_id,
    entry_type,
    amount,
    currency,
    balance_before,
    balance_after,
    idempotency_key,
    reason,
    created_at
FROM
    ledger_entries
WHERE
    request_record_id = sqlc.arg (request_record_id)
ORDER BY
    id ASC;

-- name: EnsureUserBalance :exec
INSERT INTO user_balances(user_id, currency, balance)
VALUES (sqlc.arg(user_id), sqlc.arg(currency), 0)
ON CONFLICT (user_id, currency) DO NOTHING;

-- name: AddUserBalance :one
UPDATE user_balances
SET
    balance = balance + sqlc.arg(amount),
    updated_at = now()
WHERE user_id = sqlc.arg(user_id)
AND currency = sqlc.arg(currency)
RETURNING
    id,
    user_id,
    currency,
    balance,
    created_at,
    updated_at;

-- name: SubtractUserBalance :one
UPDATE user_balances
SET
    balance = balance - sqlc.arg(amount),
    updated_at = now()
WHERE user_id = sqlc.arg(user_id)
  AND currency = sqlc.arg(currency)
  AND balance >= sqlc.arg(amount)
RETURNING
    id,
    user_id,
    currency,
    balance,
    created_at,
    updated_at;