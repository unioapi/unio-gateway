-- name: ModelExistsByID :one
SELECT EXISTS (
    SELECT 1
    FROM models m
    WHERE m.model_id = sqlc.arg(requested_model_id)
    AND m.status = 'enabled'
) AS exists;

-- name: ProjectCanUseModel :one
WITH project_scope AS (
    SELECT sqlc.arg(project_id)::BIGINT AS project_id
),
     target_model AS (
         SELECT m.id
         FROM models m
         WHERE m.model_id = sqlc.arg(requested_model_id)
           AND m.status = 'enabled'
     ),
     project_policy_mode AS (
         SELECT EXISTS (
             SELECT 1
             FROM project_model_policies pmp
                      JOIN project_scope ps ON ps.project_id = pmp.project_id
             WHERE pmp.visibility = 'allowed'
         ) AS has_allow_list
     )
SELECT EXISTS (
    SELECT 1
    FROM target_model m
             JOIN project_scope ps ON ps.project_id > 0
    WHERE NOT EXISTS (
        SELECT 1
        FROM project_model_policies denied
        WHERE denied.project_id = ps.project_id
          AND denied.model_id = m.id
          AND denied.visibility = 'denied'
    )
      AND (
        NOT (SELECT has_allow_list FROM project_policy_mode)
            OR EXISTS (
            SELECT 1
            FROM project_model_policies allowed
            WHERE allowed.project_id = ps.project_id
              AND allowed.model_id = m.id
              AND allowed.visibility = 'allowed'
        )
        )
) AS allowed;

-- name: FindRouteCandidates :many
WITH project_scope AS (
    SELECT sqlc.arg(project_id)::BIGINT AS project_id
),
     project_policy_mode AS (
         SELECT EXISTS (
             SELECT 1
             FROM project_model_policies pmp
                      JOIN project_scope ps ON ps.project_id = pmp.project_id
             WHERE pmp.visibility = 'allowed'
         ) AS has_allow_list
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
WHERE m.model_id = sqlc.arg(requested_model_id)
  AND m.status = 'enabled'
  AND cm.status = 'enabled'
  AND c.status = 'enabled'
  AND p.status = 'enabled'
  AND NOT EXISTS (
    SELECT 1
    FROM project_model_policies denied
             JOIN project_scope ps ON ps.project_id = denied.project_id
    WHERE denied.model_id = m.id
      AND denied.visibility = 'denied'
)
  AND (
    NOT (SELECT has_allow_list FROM project_policy_mode)
        OR EXISTS (
        SELECT 1
        FROM project_model_policies allowed
                 JOIN project_scope ps ON ps.project_id = allowed.project_id
        WHERE allowed.model_id = m.id
          AND allowed.visibility = 'allowed'
    )
    )
ORDER BY
    c.priority ASC,
    c.id ASC;