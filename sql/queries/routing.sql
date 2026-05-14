-- name: FindRouteCandidates :many
WITH project_scope AS (
    SELECT sqlc.arg (project_id)::BIGINT AS project_id
)
SELECT
    m.id AS model_db_id,
    m.model_id AS requested_model_id,
    p.id AS provider_id,
    p.slug AS provider_slug,
    p.adapter AS adapter_key,
    c.id AS channel_id,
    c.base_url,
    c.credential_ref,
    c.timeout_ms,
    c.priority,
    cm.upstream_model
FROM models m
JOIN channel_models cm ON cm.model_id = m.id
JOIN channels c ON c.id = cm.channel_id
JOIN providers p ON p.id = c.provider_id
JOIN project_scope ps ON ps.project_id > 0
WHERE m.model_id = sqlc.arg (requested_model_id)
  AND m.status = 'enabled'
  AND cm.status = 'enabled'
  AND c.status = 'enabled'
  AND p.status = 'enabled'
ORDER BY
    c.priority ASC,
    c.id ASC;