-- name: CreateUsageRecord :one
INSERT INTO usage_records (
    request_record_id,
    prompt_tokens,
    completion_tokens,
    total_tokens,
    cached_tokens,
    reasoning_tokens,
    source
)
VALUES (
           sqlc.arg(request_record_id),
           sqlc.arg(prompt_tokens),
           sqlc.arg(completion_tokens),
           sqlc.arg(total_tokens),
           sqlc.arg(cached_tokens),
           sqlc.arg(reasoning_tokens),
           sqlc.arg(source)
       )
RETURNING
    id,
    request_record_id,
    prompt_tokens,
    completion_tokens,
    total_tokens,
    cached_tokens,
    reasoning_tokens,
    source,
    created_at;

-- name: GetUsageRecordByRequest :one
SELECT
    id,
    request_record_id,
    prompt_tokens,
    completion_tokens,
    total_tokens,
    cached_tokens,
    reasoning_tokens,
    source,
    created_at
FROM usage_records
WHERE request_record_id = sqlc.arg(request_record_id);