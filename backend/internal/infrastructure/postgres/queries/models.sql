-- name: ListModels :many
SELECT * FROM models
WHERE is_active = true
ORDER BY provider_kind, model_slug;

-- name: ListModelsByProvider :many
SELECT * FROM models
WHERE provider_kind = $1 AND is_active = true
ORDER BY model_slug;