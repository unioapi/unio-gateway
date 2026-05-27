-- name: CreateProject :one
-- CreateProject 在指定用户下创建项目。
INSERT INTO projects (user_id, name)
VALUES ($1, $2)
RETURNING id, user_id, name, created_at, updated_at;

-- name: GetProjectForUser :one
-- GetProjectForUser 按 project_id 和 user_id 读取项目并校验归属。
SELECT id, user_id, name, created_at, updated_at
FROM projects
WHERE id = sqlc.arg(project_id)
AND user_id = sqlc.arg(user_id)
LIMIT 1;
