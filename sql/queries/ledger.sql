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
    reserved_balance,
    created_at,
    updated_at;

-- name: GetUserBalance :one
SELECT
    id,
    user_id,
    currency,
    balance,
    reserved_balance,
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
    reserved_balance,
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
    reserved_balance,
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
    id;

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
    reserved_balance,
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
  AND balance - reserved_balance >= sqlc.arg(amount)
RETURNING
    id,
    user_id,
    currency,
    balance,
    reserved_balance,
    created_at,
    updated_at;

-- name: ReserveUserBalance :one
UPDATE user_balances
SET
    reserved_balance = reserved_balance + sqlc.arg(amount),
    updated_at = now()
WHERE user_id = sqlc.arg(user_id)
    AND currency = sqlc.arg(currency)
    AND balance - reserved_balance >= sqlc.arg(amount)
RETURNING
    id,
    user_id,
    currency,
    balance,
    reserved_balance,
    created_at,
    updated_at;

-- name: CreateLedgerReservation :one
INSERT INTO ledger_reservations (
    user_id,
    request_record_id,
    currency,
    status,
    authorized_amount,
    idempotency_key,
    reason
)
VALUES (
           sqlc.arg(user_id),
           sqlc.arg(request_record_id),
           sqlc.arg(currency),
           'authorized',
           sqlc.arg(authorized_amount),
           sqlc.arg(idempotency_key),
           sqlc.arg(reason)
       )
RETURNING
    id,
    user_id,
    request_record_id,
    currency,
    status,
    authorized_amount,
    captured_amount,
    released_amount,
    capture_ledger_entry_id,
    idempotency_key,
    reason,
    created_at,
    updated_at,
    captured_at,
    released_at;

-- name: GetLedgerReservationByIdempotencyKey :one
SELECT
    id,
    user_id,
    request_record_id,
    currency,
    status,
    authorized_amount,
    captured_amount,
    released_amount,
    capture_ledger_entry_id,
    idempotency_key,
    reason,
    created_at,
    updated_at,
    captured_at,
    released_at
FROM ledger_reservations
WHERE idempotency_key = sqlc.arg(idempotency_key);

-- name: GetLedgerReservationByRequestRecordID :one
SELECT
    id,
    user_id,
    request_record_id,
    currency,
    status,
    authorized_amount,
    captured_amount,
    released_amount,
    capture_ledger_entry_id,
    idempotency_key,
    reason,
    created_at,
    updated_at,
    captured_at,
    released_at
FROM ledger_reservations
WHERE request_record_id = sqlc.arg(request_record_id);

-- name: GetLedgerReservationByRequestRecordIDForUpdate :one
SELECT
    id,
    user_id,
    request_record_id,
    currency,
    status,
    authorized_amount,
    captured_amount,
    released_amount,
    capture_ledger_entry_id,
    idempotency_key,
    reason,
    created_at,
    updated_at,
    captured_at,
    released_at
FROM ledger_reservations
WHERE request_record_id = sqlc.arg(request_record_id)
    FOR UPDATE;

-- name: CaptureUserReservedBalance :one
UPDATE user_balances
SET
    balance = balance - sqlc.arg(captured_amount),
    reserved_balance = reserved_balance - sqlc.arg(authorized_amount),
    updated_at = now()
WHERE user_id = sqlc.arg(user_id)
  AND currency = sqlc.arg(currency)
  AND reserved_balance >= sqlc.arg(authorized_amount)
  AND balance >= sqlc.arg(captured_amount)
RETURNING
    id,
    user_id,
    currency,
    balance,
    reserved_balance,
    created_at,
    updated_at;

-- name: ReleaseUserReservedBalance :one
UPDATE user_balances
SET
    reserved_balance = reserved_balance - sqlc.arg(amount),
    updated_at = now()
WHERE user_id = sqlc.arg(user_id)
  AND currency = sqlc.arg(currency)
  AND reserved_balance >= sqlc.arg(amount)
RETURNING
    id,
    user_id,
    currency,
    balance,
    reserved_balance,
    created_at,
    updated_at;

-- name: CaptureLedgerReservation :one
UPDATE ledger_reservations
SET
    status = 'captured',
    captured_amount = sqlc.arg(captured_amount),
    released_amount = authorized_amount - sqlc.arg(captured_amount),
    capture_ledger_entry_id = sqlc.arg(capture_ledger_entry_id),
    captured_at = now(),
    released_at = CASE
                      WHEN authorized_amount > sqlc.arg(captured_amount) THEN now()
                      ELSE NULL
        END,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND status = 'authorized'
  AND authorized_amount >= sqlc.arg(captured_amount)
RETURNING
    id,
    user_id,
    request_record_id,
    currency,
    status,
    authorized_amount,
    captured_amount,
    released_amount,
    capture_ledger_entry_id,
    idempotency_key,
    reason,
    created_at,
    updated_at,
    captured_at,
    released_at;

-- name: ReleaseLedgerReservation :one
UPDATE ledger_reservations
SET
    status = 'released',
    released_amount = authorized_amount,
    released_at = now(),
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND status = 'authorized'
RETURNING
    id,
    user_id,
    request_record_id,
    currency,
    status,
    authorized_amount,
    captured_amount,
    released_amount,
    capture_ledger_entry_id,
    idempotency_key,
    reason,
    created_at,
    updated_at,
    captured_at,
    released_at;