-- name: CreateUser :one
-- CreateUser 创建用户账号并返回用户事实。
INSERT INTO users (email, password_hash, display_name)
VALUES ($1, $2, $3)
RETURNING id, email, password_hash, display_name, created_at, updated_at;

-- name: GetUserByEmail :one
-- GetUserByEmail 按邮箱大小写不敏感读取用户账号。
SELECT id, email, password_hash, display_name, created_at, updated_at
FROM users
WHERE lower(email) = lower(sqlc.arg(email))
LIMIT 1;
