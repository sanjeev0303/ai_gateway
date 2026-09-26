-- name: CreatePlan :one
INSERT INTO plans (code, name, description, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListPlans :many
SELECT * FROM plans
ORDER BY created_at;

-- name: CreateSubscription :one
INSERT INTO subscriptions (organization_id, plan_id, status, current_period_start, current_period_end, cancel_at_period_end)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetSubscription :one
SELECT * FROM subscriptions
WHERE organization_id = $1 AND id = $2;

-- name: ListSubscriptions :many
SELECT * FROM subscriptions
WHERE organization_id = $1
ORDER BY created_at;

-- name: RecordBillingEvent :one
INSERT INTO billing_events (organization_id, billing_event_id, event_type, amount_micros, currency, status, payload)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
ON CONFLICT (organization_id, billing_event_id) DO NOTHING
RETURNING *;

-- name: GetBillingEvent :one
SELECT * FROM billing_events
WHERE organization_id = $1 AND billing_event_id = $2;