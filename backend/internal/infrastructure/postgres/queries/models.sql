-- name: ListModels :many
SELECT * FROM models
WHERE is_active = true
ORDER BY provider_kind, model_slug;

-- name: ListModelsByProvider :many
SELECT * FROM models
WHERE provider_kind = $1 AND is_active = true
ORDER BY model_slug;

-- name: UpsertModel :one
INSERT INTO models (provider_kind, model_slug, display_name, is_active)
VALUES ($1, $2, $3, $4)
ON CONFLICT (provider_kind, model_slug) DO UPDATE
SET display_name = EXCLUDED.display_name,
    is_active    = EXCLUDED.is_active
RETURNING *;
