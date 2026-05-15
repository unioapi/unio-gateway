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
    started_at,
    completed_at,
    created_at,
    updated_at;

-- name: MarkRequestRunning :one
UPDATE request_records
SET status = 'running',
    updated_at = now()
WHERE id = sqlc.arg(id)
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
    started_at,
    completed_at,
    created_at,
    updated_at;

-- name: MarkRequestSucceeded :one
UPDATE request_records
SET status = 'succeeded',
    response_model_id = sqlc.arg(response_model_id),
    final_provider_id = sqlc.arg(final_provider_id),
    final_channel_id = sqlc.arg(final_channel_id),
    completed_at = sqlc.arg(completed_at),
    updated_at = now()
WHERE id = sqlc.arg(id)
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
    started_at,
    completed_at,
    created_at,
    updated_at;

-- name: MarkRequestFailed :one
UPDATE request_records
SET status = 'failed',
    error_code = sqlc.arg(error_code),
    error_message = sqlc.arg(error_message),
    completed_at = sqlc.arg(completed_at),
    updated_at = now()
WHERE id = sqlc.arg(id)
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
    started_at,
    completed_at,
    created_at,
    updated_at;

-- name: MarkRequestCanceled :one
UPDATE request_records
SET status = 'canceled',
    error_code = sqlc.arg(error_code),
    error_message = sqlc.arg(error_message),
    completed_at = sqlc.arg(completed_at),
    updated_at = now()
WHERE id = sqlc.arg(id)
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
    started_at,
    completed_at,
    created_at,
    updated_at;

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
    started_at,
    completed_at,
    created_at;

-- name: MarkRequestAttemptSucceeded :one
UPDATE request_attempts
SET status = 'succeeded',
    upstream_response_model = sqlc.arg(upstream_response_model),
    upstream_status_code = sqlc.arg(upstream_status_code),
    upstream_request_id = sqlc.arg(upstream_request_id),
    completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id)
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
    started_at,
    completed_at,
    created_at;

-- name: MarkRequestAttemptFailed :one
UPDATE request_attempts
SET status = 'failed',
    upstream_status_code = sqlc.arg(upstream_status_code),
    upstream_request_id = sqlc.arg(upstream_request_id),
    error_code = sqlc.arg(error_code),
    error_message = sqlc.arg(error_message),
    completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id)
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
    started_at,
    completed_at,
    created_at;

-- name: MarkRequestAttemptCanceled :one
UPDATE request_attempts
SET status = 'canceled',
    error_code = sqlc.arg(error_code),
    error_message = sqlc.arg(error_message),
    completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id)
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
    started_at,
    completed_at,
    created_at;

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
    started_at,
    completed_at,
    created_at
FROM request_attempts
WHERE request_record_id = sqlc.arg(request_record_id)
ORDER BY attempt_index;
