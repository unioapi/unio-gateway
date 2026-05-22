-- ledger_reservations 表示一次请求的余额预授权事实。
CREATE TABLE ledger_reservations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id),
    request_record_id BIGINT NOT NULL,
    currency TEXT NOT NULL CHECK (currency <> ''),
    status TEXT NOT NULL CHECK (status IN ('authorized', 'captured', 'released')),
    authorized_amount NUMERIC(20, 10) NOT NULL CHECK (authorized_amount > 0),
    captured_amount NUMERIC(20, 10) NOT NULL DEFAULT 0 CHECK (captured_amount >= 0),
    released_amount NUMERIC(20, 10) NOT NULL DEFAULT 0 CHECK (released_amount >= 0),
    capture_ledger_entry_id BIGINT UNIQUE,
    idempotency_key TEXT NOT NULL UNIQUE CHECK (idempotency_key <> ''),
    reason TEXT NOT NULL CHECK (reason <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    captured_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    UNIQUE (request_record_id),
    CONSTRAINT fk_ledger_reservations_request_user FOREIGN KEY (request_record_id, user_id) REFERENCES request_records (id, user_id),
    CONSTRAINT ck_ledger_reservations_amount_sum CHECK (
     captured_amount + released_amount <= authorized_amount
     ),
    CONSTRAINT ck_ledger_reservations_status_amounts CHECK (
     (
         status = 'authorized'
             AND captured_amount = 0
             AND released_amount = 0
             AND capture_ledger_entry_id IS NULL
             AND captured_at IS NULL
             AND released_at IS NULL
         )
         OR (
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
                 OR (
                 released_amount > 0
                     AND released_at IS NOT NULL
                 )
             )
         )
         OR (
         status = 'released'
             AND captured_amount = 0
             AND released_amount = authorized_amount
             AND capture_ledger_entry_id IS NULL
             AND captured_at IS NULL
             AND released_at IS NOT NULL
         )
     ),

    CONSTRAINT fk_ledger_reservations_capture_entry
        FOREIGN KEY (capture_ledger_entry_id, user_id, request_record_id)
            REFERENCES ledger_entries (id, user_id, request_record_id)
);

CREATE INDEX idx_ledger_reservations_user_created_at
    ON ledger_reservations(user_id, created_at DESC, id DESC);

CREATE INDEX idx_ledger_reservations_authorized_created_at
    ON ledger_reservations(created_at, id)
    WHERE status = 'authorized';