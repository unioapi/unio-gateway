-- name: CreatePrice :one
-- CreatePrice 创建模型客户侧售卖价配置。
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
-- FindActivePriceForModel 查找指定模型在指定时间生效的客户侧售卖价。
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
