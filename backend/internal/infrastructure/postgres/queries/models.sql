-- name: ListModels :many
SELECT * FROM models
WHERE is_active = true
ORDER BY provider_kind, model_slug;

-- name: ListAllModels :many
SELECT * FROM models
ORDER BY provider_kind, model_slug;

-- name: ListModelsByProvider :many
SELECT * FROM models
WHERE provider_kind = $1 AND is_active = true
ORDER BY model_slug;

-- name: GetModelBySlug :one
SELECT * FROM models
WHERE provider_kind = $1 AND model_slug = $2;

-- name: UpsertModel :one
INSERT INTO models (provider_kind, model_slug, display_name, is_active)
VALUES ($1, $2, $3, $4)
ON CONFLICT (provider_kind, model_slug) DO UPDATE
SET display_name = EXCLUDED.display_name,
    is_active    = EXCLUDED.is_active,
    updated_at   = now()
RETURNING *;

-- name: UpsertModelWithMetadata :one
INSERT INTO models (
    provider_kind, model_slug, display_name, is_active,
    capabilities, context_limit, supports_streaming, supports_tools, supports_reasoning,
    status, pricing_version
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8, $9,
    $10, $11
) ON CONFLICT (provider_kind, model_slug) DO UPDATE
SET display_name       = EXCLUDED.display_name,
    is_active          = EXCLUDED.is_active,
    capabilities       = EXCLUDED.capabilities,
    context_limit      = EXCLUDED.context_limit,
    supports_streaming = EXCLUDED.supports_streaming,
    supports_tools     = EXCLUDED.supports_tools,
    supports_reasoning = EXCLUDED.supports_reasoning,
    status             = EXCLUDED.status,
    pricing_version    = EXCLUDED.pricing_version,
    updated_at         = now()
RETURNING *;

-- name: MarkModelsInactiveExcept :exec
UPDATE models
SET is_active = false, status = 'inactive', updated_at = now()
WHERE provider_kind = $1 AND NOT (model_slug = ANY($2::text[]));
