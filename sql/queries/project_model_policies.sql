-- name: ProjectCanUseModel :one
-- ProjectCanUseModel 判断指定项目是否允许使用指定启用模型。
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
