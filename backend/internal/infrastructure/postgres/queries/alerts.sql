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

-- name: UpdateAlertTriggered :one
UPDATE alerts
SET last_triggered_at = now(),
    status = $3,
    updated_at = now()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: CreateAlertEvent :one
INSERT INTO alert_events (organization_id, alert_id, alert_type, severity, status, message, context, triggered_at, resolved_at)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9)
RETURNING *;

-- name: ListAlertEvents :many
SELECT * FROM alert_events
WHERE organization_id = $1
  AND (sqlc.narg(alert_id)::uuid IS NULL OR alert_id = sqlc.narg(alert_id))
ORDER BY triggered_at DESC
LIMIT $2 OFFSET $3;

-- name: GetLatestAlertEvent :one
SELECT * FROM alert_events
WHERE organization_id = $1 AND alert_id = $2
ORDER BY triggered_at DESC
LIMIT 1;