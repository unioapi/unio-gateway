-- name: CreatePrice :one
INSERT INTO prices (
    model_id,
    currency,
    pricing_unit,
    input_price,
    output_price,
    cached_input_price,
    reasoning_output_price,
    status,
    effective_from,
    effective_to
)
VALUES (
    sqlc.arg(model_id),
    sqlc.arg(currency),
    sqlc.arg(pricing_unit),
    sqlc.arg(input_price),
    sqlc.arg(output_price),
    sqlc.arg(cached_input_price),
    sqlc.arg(reasoning_output_price),
    sqlc.arg(status),
    sqlc.arg(effective_from),
    sqlc.arg(effective_to)
)
RETURNING
    id,
    model_id,
    currency,
    pricing_unit,
    input_price,
    output_price,
    cached_input_price,
    reasoning_output_price,
    status,
    effective_from,
    effective_to,
    created_at,
    updated_at;

-- name: FindActivePriceForModel :one
SELECT
    id,
    model_id,
    currency,
    pricing_unit,
    input_price,
    output_price,
    cached_input_price,
    reasoning_output_price,
    status,
    effective_from,
    effective_to,
    created_at,
    updated_at
FROM prices
WHERE model_id = sqlc.arg(model_id)
    AND status = 'enabled'
    AND effective_from <= sqlc.arg(at_time)
    AND (
        effective_to IS NULL
        OR effective_to > sqlc.arg(at_time)
    )
ORDER BY effective_from DESC, id DESC
LIMIT 1;

-- name: CreatePriceSnapshot :one
INSERT INTO price_snapshots (
    request_record_id,
    price_id,
    currency,
    pricing_unit,
    input_price,
    output_price,
    cached_input_price,
    reasoning_output_price,
    formula_version
)
VALUES (
           sqlc.arg(request_record_id),
           sqlc.arg(price_id),
           sqlc.arg(currency),
           sqlc.arg(pricing_unit),
           sqlc.arg(input_price),
           sqlc.arg(output_price),
           sqlc.arg(cached_input_price),
           sqlc.arg(reasoning_output_price),
           sqlc.arg(formula_version)
       )
RETURNING
    id,
    request_record_id,
    price_id,
    currency,
    pricing_unit,
    input_price,
    output_price,
    cached_input_price,
    reasoning_output_price,
    formula_version,
    created_at;

-- name: GetPriceSnapshotByRequest :one
SELECT
    id,
    request_record_id,
    price_id,
    currency,
    pricing_unit,
    input_price,
    output_price,
    cached_input_price,
    reasoning_output_price,
    formula_version,
    created_at
FROM price_snapshots
WHERE request_record_id = sqlc.arg(request_record_id);
