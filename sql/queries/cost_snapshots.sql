-- name: CreateCostSnapshot :one
-- CreateCostSnapshot 创建一次请求结算使用的上游成本价快照和实际成本事实。
INSERT INTO cost_snapshots (
    request_record_id,
    cost_price_id,
    provider_id,
    channel_id,
    model_id,
    upstream_model,
    currency,
    pricing_unit,
    input_cost,
    output_cost,
    cached_input_cost,
    reasoning_output_cost,
    input_cost_amount,
    output_cost_amount,
    cached_input_cost_amount,
    reasoning_output_cost_amount,
    total_cost_amount,
    formula_version
)
VALUES (
    sqlc.arg(request_record_id),
    sqlc.arg(cost_price_id),
    sqlc.arg(provider_id),
    sqlc.arg(channel_id),
    sqlc.arg(model_id),
    sqlc.arg(upstream_model),
    sqlc.arg(currency),
    sqlc.arg(pricing_unit),
    sqlc.arg(input_cost),
    sqlc.arg(output_cost),
    sqlc.arg(cached_input_cost),
    sqlc.arg(reasoning_output_cost),
    sqlc.arg(input_cost_amount),
    sqlc.arg(output_cost_amount),
    sqlc.arg(cached_input_cost_amount),
    sqlc.arg(reasoning_output_cost_amount),
    sqlc.arg(total_cost_amount),
    sqlc.arg(formula_version)
)
RETURNING
    id,
    request_record_id,
    cost_price_id,
    provider_id,
    channel_id,
    model_id,
    upstream_model,
    currency,
    pricing_unit,
    input_cost,
    output_cost,
    cached_input_cost,
    reasoning_output_cost,
    input_cost_amount,
    output_cost_amount,
    cached_input_cost_amount,
    reasoning_output_cost_amount,
    total_cost_amount,
    formula_version,
    created_at;

-- name: GetCostSnapshotByRequest :one
-- GetCostSnapshotByRequest 按请求 ID 读取上游成本快照。
SELECT
    id,
    request_record_id,
    cost_price_id,
    provider_id,
    channel_id,
    model_id,
    upstream_model,
    currency,
    pricing_unit,
    input_cost,
    output_cost,
    cached_input_cost,
    reasoning_output_cost,
    input_cost_amount,
    output_cost_amount,
    cached_input_cost_amount,
    reasoning_output_cost_amount,
    total_cost_amount,
    formula_version,
    created_at
FROM cost_snapshots
WHERE request_record_id = sqlc.arg(request_record_id);
