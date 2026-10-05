-- name: CreateAlert :one
INSERT INTO alerts (organization_id, alert_type, name, severity, configuration, status)
VALUES ($1, $2, $3, $4, $5::jsonb, $6)
RETURNING *;

-- name: ListAlerts :many
SELECT * FROM alerts
WHERE organization_id = $1
ORDER BY created_at;

-- name: GetAlert :one
SELECT * FROM alerts
WHERE organization_id = $1 AND id = $2;

-- name: UpdateAlert :one
UPDATE alerts
SET name          = $3,
    severity      = $4,
    configuration = $5::jsonb,
    status        = $6,
    updated_at    = now()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: DeleteAlert :execrows
DELETE FROM alerts
WHERE organization_id = $1 AND id = $2;