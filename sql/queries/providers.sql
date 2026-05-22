-- name: ListEnabledProviderAdapters :many
SELECT
    id,
    slug,
    adapter
FROM providers
WHERE status = 'enabled'
ORDER BY slug;
