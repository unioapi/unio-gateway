-- name: CreateRequestRecord :one
INSERT INTO request_records (
    request_id,
    user_id,
    project_id,
    api_key_id,
    requested_model_id,
    response_model_id,
    stream,
    status,
    final_provider_id,
    final_channel_id,
    error_code,
    error_message,
    internal_error_detail,
    started_at,
    completed_at
)
VALUES (
           sqlc.arg(request_id),
           sqlc.arg(user_id),
           sqlc.arg(project_id),
           sqlc.arg(api_key_id),
           sqlc.arg(requested_model_id),
           sqlc.arg(response_model_id),
           sqlc.arg(stream),
           sqlc.arg(status),
           sqlc.arg(final_provider_id),
           sqlc.arg(final_channel_id),
           sqlc.arg(error_code),
           sqlc.arg(error_message),
           sqlc.arg(internal_error_detail),
           sqlc.arg(started_at),
           sqlc.arg(completed_at)
       )
RETURNING
    id,
    request_id,
    user_id,
    project_id,
    api_key_id,
    requested_model_id,
    response_model_id,
    stream,
    status,
    final_provider_id,
    final_channel_id,
    error_code,
    error_message,
    internal_error_detail,
    started_at,
    completed_at,
    created_at,
    updated_at;

-- name: GetRequestRecordForUpdate :one
-- settlement 入口先锁 request 行，串行化同一个 request 的并发结算。
-- running 表示本次可以继续首次 settlement；
-- succeeded 表示可能是幂等重放，需要检查既有 usage/snapshot/ledger 后直接返回。
SELECT
    id,
    request_id,
    user_id,
    project_id,
    api_key_id,
    requested_model_id,
    response_model_id,
    stream,
    status,
    final_provider_id,
    final_channel_id,
    error_code,
    error_message,
    internal_error_detail,
    started_at,
    completed_at,
    created_at,
    updated_at
FROM request_records
WHERE id = sqlc.arg(request_record_id)
    FOR UPDATE;

-- name: MarkRequestRunning :one
-- request 状态机由 SQL 原子守卫：pending 才能进入 running。
-- 如果已经是 running，说明调用方重复推进同一阶段，直接返回原事实。
-- succeeded/failed/canceled 是终态，不会被重新打开。
WITH updated AS (
    UPDATE request_records
        SET status = 'running',
            updated_at = now()
        WHERE request_records.id = sqlc.arg(request_record_id)
            AND request_records.status = 'pending'
        RETURNING request_records.*
)
SELECT *
FROM updated

UNION ALL

SELECT request_records.*
FROM request_records
WHERE request_records.id = sqlc.arg(request_record_id)
  AND request_records.status = 'running'
  AND NOT EXISTS (SELECT 1 FROM updated);

-- name: MarkRequestSucceeded :one
-- running 才能进入 succeeded。
-- 如果已经 succeeded，重复 settlement 只能读回第一次成功事实，不能覆盖 response_model/provider/channel/completed_at。
WITH updated AS (
    UPDATE request_records
        SET status = 'succeeded',
            response_model_id = sqlc.arg(response_model_id),
            final_provider_id = sqlc.arg(final_provider_id),
            final_channel_id = sqlc.arg(final_channel_id),
            completed_at = sqlc.arg(completed_at),
            updated_at = now()
        WHERE request_records.id = sqlc.arg(request_record_id)
            AND request_records.status = 'running'
        RETURNING request_records.*
)
SELECT *
FROM updated

UNION ALL

SELECT request_records.*
FROM request_records
WHERE request_records.id = sqlc.arg(request_record_id)
  AND request_records.status = 'succeeded'
  AND NOT EXISTS (SELECT 1 FROM updated);

-- name: MarkRequestFailed :one
-- running 才能进入 failed。
-- 如果已经 failed，重复失败写入只返回第一次失败事实，不能覆盖 error_code/error_message/internal_error_detail/completed_at。
WITH updated AS (
    UPDATE request_records
        SET status = 'failed',
            error_code = sqlc.arg(error_code),
            error_message = sqlc.arg(error_message),
            internal_error_detail = sqlc.arg(internal_error_detail),
            completed_at = sqlc.arg(completed_at),
            updated_at = now()
        WHERE request_records.id = sqlc.arg(request_record_id)
            AND request_records.status = 'running'
        RETURNING request_records.*
)
SELECT *
FROM updated

UNION ALL

SELECT request_records.*
FROM request_records
WHERE request_records.id = sqlc.arg(request_record_id)
  AND request_records.status = 'failed'
  AND NOT EXISTS (SELECT 1 FROM updated);

-- name: MarkRequestCanceled :one
-- running 才能进入 canceled。
-- 如果已经 canceled，重复取消写入只返回第一次取消事实，不能覆盖 error_code/error_message/internal_error_detail/completed_at。
WITH updated AS (
    UPDATE request_records
        SET status = 'canceled',
            error_code = sqlc.arg(error_code),
            error_message = sqlc.arg(error_message),
            internal_error_detail = sqlc.arg(internal_error_detail),
            completed_at = sqlc.arg(completed_at),
            updated_at = now()
        WHERE request_records.id = sqlc.arg(request_record_id)
            AND request_records.status = 'running'
        RETURNING request_records.*
)
SELECT *
FROM updated

UNION ALL

SELECT request_records.*
FROM request_records
WHERE request_records.id = sqlc.arg(request_record_id)
  AND request_records.status = 'canceled'
  AND NOT EXISTS (SELECT 1 FROM updated);

-- name: CreateRequestAttempt :one
INSERT INTO request_attempts (
    request_record_id,
    attempt_index,
    provider_id,
    channel_id,
    adapter_key,
    upstream_model,
    upstream_response_model,
    status,
    upstream_status_code,
    upstream_request_id,
    error_code,
    error_message,
    internal_error_detail,
    started_at,
    completed_at
)
VALUES (
           sqlc.arg(request_record_id),
           sqlc.arg(attempt_index),
           sqlc.arg(provider_id),
           sqlc.arg(channel_id),
           sqlc.arg(adapter_key),
           sqlc.arg(upstream_model),
           sqlc.arg(upstream_response_model),
           sqlc.arg(status),
           sqlc.arg(upstream_status_code),
           sqlc.arg(upstream_request_id),
           sqlc.arg(error_code),
           sqlc.arg(error_message),
           sqlc.arg(internal_error_detail),
           sqlc.arg(started_at),
           sqlc.arg(completed_at)
       )
RETURNING
    id,
    request_record_id,
    attempt_index,
    provider_id,
    channel_id,
    adapter_key,
    upstream_model,
    upstream_response_model,
    status,
    upstream_status_code,
    upstream_request_id,
    error_code,
    error_message,
    internal_error_detail,
    started_at,
    completed_at,
    created_at;

-- name: MarkRequestAttemptSucceeded :one
-- running 才能进入 succeeded。
-- 如果已经 succeeded，重复成功写入只返回第一次成功事实，不能覆盖 upstream response metadata。
WITH updated AS (
    UPDATE request_attempts
        SET status = 'succeeded',
            upstream_response_model = sqlc.arg(upstream_response_model),
            upstream_status_code = sqlc.arg(upstream_status_code),
            upstream_request_id = sqlc.arg(upstream_request_id),
            completed_at = sqlc.arg(completed_at)
        WHERE request_attempts.id = sqlc.arg(attempt_id)
            AND request_attempts.status = 'running'
        RETURNING request_attempts.*
)
SELECT *
FROM updated

UNION ALL

SELECT request_attempts.*
FROM request_attempts
WHERE request_attempts.id = sqlc.arg(attempt_id)
  AND request_attempts.status = 'succeeded'
  AND NOT EXISTS (SELECT 1 FROM updated);

-- name: MarkRequestAttemptFailed :one
-- running 才能进入 failed。
-- 如果已经 failed，重复失败写入只返回第一次失败事实，不能覆盖 error/upstream metadata。
WITH updated AS (
    UPDATE request_attempts
        SET status = 'failed',
            upstream_status_code = sqlc.arg(upstream_status_code),
            upstream_request_id = sqlc.arg(upstream_request_id),
            error_code = sqlc.arg(error_code),
            error_message = sqlc.arg(error_message),
            internal_error_detail = sqlc.arg(internal_error_detail),
            completed_at = sqlc.arg(completed_at)
        WHERE request_attempts.id = sqlc.arg(attempt_id)
            AND request_attempts.status = 'running'
        RETURNING request_attempts.*
)
SELECT *
FROM updated

UNION ALL

SELECT request_attempts.*
FROM request_attempts
WHERE request_attempts.id = sqlc.arg(attempt_id)
  AND request_attempts.status = 'failed'
  AND NOT EXISTS (SELECT 1 FROM updated);

-- name: MarkRequestAttemptCanceled :one
-- running 才能进入 canceled。
-- 如果已经 canceled，重复取消写入只返回第一次取消事实，不能覆盖 error metadata。
WITH updated AS (
    UPDATE request_attempts
        SET status = 'canceled',
            error_code = sqlc.arg(error_code),
            error_message = sqlc.arg(error_message),
            internal_error_detail = sqlc.arg(internal_error_detail),
            completed_at = sqlc.arg(completed_at)
        WHERE request_attempts.id = sqlc.arg(attempt_id)
            AND request_attempts.status = 'running'
        RETURNING request_attempts.*
)
SELECT *
FROM updated

UNION ALL

SELECT request_attempts.*
FROM request_attempts
WHERE request_attempts.id = sqlc.arg(attempt_id)
  AND request_attempts.status = 'canceled'
  AND NOT EXISTS (SELECT 1 FROM updated);

-- name: ListRequestAttemptsByRequest :many
SELECT
    id,
    request_record_id,
    attempt_index,
    provider_id,
    channel_id,
    adapter_key,
    upstream_model,
    upstream_response_model,
    status,
    upstream_status_code,
    upstream_request_id,
    error_code,
    error_message,
    internal_error_detail,
    started_at,
    completed_at,
    created_at
FROM request_attempts
WHERE request_record_id = sqlc.arg(request_record_id)
ORDER BY attempt_index;
