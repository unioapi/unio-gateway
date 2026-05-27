-- name: ListEnabledProviderAdapters :many
-- ListEnabledProviderAdapters 列出启用 provider 的 adapter 注册键。
SELECT
    id,
    slug,
    adapter
FROM providers
WHERE status = 'enabled'
ORDER BY slug;
