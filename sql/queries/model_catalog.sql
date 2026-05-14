-- name: ListAvailableModelsForProject :many
WITH project_scope AS (
    SELECT sqlc.arg (project_id)::BIGINT AS project_id
)
SELECT DISTINCT
    m.id,
    m.model_id,
    m.display_name,
    m.owned_by
FROM models m
JOIN channel_models cm ON cm.model_id = m.id
JOIN channels c ON c.id = cm.channel_id
JOIN providers p ON p.id = c.provider_id
JOIN project_scope ps ON ps.project_id > 0
WHERE m.status = 'enabled'
    AND cm.status = 'enabled'
    AND c.status = 'enabled'
    AND p.status = 'enabled'
ORDER BY m.model_id ASC;