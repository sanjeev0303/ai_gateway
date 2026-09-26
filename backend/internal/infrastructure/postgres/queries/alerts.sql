-- name: CreateAlert :one
INSERT INTO alerts (organization_id, alert_type, name, severity, configuration, status)
VALUES ($1, $2, $3, $4, $5::jsonb, $6)
RETURNING *;

-- name: ListAlerts :many
SELECT * FROM alerts
WHERE organization_id = $1
ORDER BY created_at;

-- name: SetAlertStatus :one
UPDATE alerts
SET status = $3
WHERE organization_id = $1 AND id = $2
RETURNING *;