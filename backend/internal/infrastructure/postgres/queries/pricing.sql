-- name: UpsertPricingModel :one
INSERT INTO pricing_models (organization_id, model_slug, provider_kind, display_name, currency, status)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (organization_id, model_slug) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    status       = EXCLUDED.status,
    updated_at   = now()
RETURNING *;

-- name: GetPricingModel :one
SELECT * FROM pricing_models
WHERE organization_id = $1 AND id = $2;

-- name: GetPricingModelBySlug :one
SELECT * FROM pricing_models
WHERE organization_id = $1 AND model_slug = $2;

-- name: ListPricingModels :many
SELECT * FROM pricing_models
WHERE organization_id = $1
ORDER BY created_at;

-- name: CreatePricingVersion :one
INSERT INTO pricing_versions (
    pricing_model_id, organization_id, version,
    input_unit_price_micros, output_unit_price_micros, cache_read_input_token_price_micros,
    effective_from, effective_to
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetPricingVersion :one
SELECT * FROM pricing_versions
WHERE organization_id = $1 AND id = $2;

-- name: GetPricingVersionByNumber :one
SELECT * FROM pricing_versions
WHERE organization_id = $1 AND pricing_model_id = $2 AND version = $3;

-- name: GetLatestPricingVersion :one
SELECT * FROM pricing_versions
WHERE organization_id = $1 AND pricing_model_id = $2
ORDER BY version DESC
LIMIT 1;

-- name: GetEffectivePricingVersion :one
SELECT * FROM pricing_versions
WHERE organization_id = $1 AND pricing_model_id = $2
  AND effective_from <= $3
  AND (effective_to IS NULL OR effective_to > $3)
ORDER BY effective_from DESC, version DESC
LIMIT 1;

-- name: ListPricingVersions :many
SELECT * FROM pricing_versions
WHERE organization_id = $1 AND pricing_model_id = $2
ORDER BY effective_from DESC, version DESC;