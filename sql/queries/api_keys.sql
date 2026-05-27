-- name: CreateAPIKey :one
-- CreateAPIKey 创建项目 API Key，只保存 key hash 和展示前缀。
INSERT INTO api_keys (project_id, name, key_prefix, key_hash, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, project_id, name, key_prefix, key_hash, last_used_at, expires_at, disabled_at, revoked_at, created_at, updated_at;

-- name: GetAPIKeyByHash :one
-- GetAPIKeyByHash 按 key hash 读取 API Key，并带出所属用户 ID。
SELECT k.id, p.user_id, k.project_id, k.name, k.key_prefix, k.key_hash, k.last_used_at, k.expires_at, k.disabled_at, k.revoked_at, k.created_at, k.updated_at
FROM api_keys k
JOIN projects p ON p.id = k.project_id
WHERE key_hash = $1
LIMIT 1;

-- name: UpdateAPIKeyLastUsedAt :exec
-- UpdateAPIKeyLastUsedAt 更新 API Key 最近使用时间。
UPDATE api_keys
SET last_used_at = sqlc.arg(last_used_at), updated_at = now()
WHERE id = sqlc.arg(id);
