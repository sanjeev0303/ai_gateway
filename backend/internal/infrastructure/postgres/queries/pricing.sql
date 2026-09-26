-- name: UpsertPricingModel :one
INSERT INTO pricing_models (organization_id, model_slug, provider_kind, display_name, currency, status)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (organization_id, model_slug) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    status       = EXCLUDED.status
RETURNING *;

-- name: ListPricingModels :many
SELECT * FROM pricing_models
WHERE organization_id = $1
ORDER BY created_at;

-- name: CreatePricingVersion :one
INSERT INTO pricing_versions (pricing_model_id, organization_id, version, input_unit_price_micros, output_unit_price_micros, effective_from)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListPricingVersions :many
SELECT * FROM pricing_versions
WHERE organization_id = $1 AND pricing_model_id = $2
ORDER BY effective_from DESC;