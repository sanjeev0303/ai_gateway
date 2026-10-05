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
-- The organization's current plan: its active (or trialing) subscription
-- joined to the plan catalog. Filters to the same statuses as
-- subscriptions_one_active_per_org, so at most one row is ever returned.
-- Columns are aliased from both tables because both carry id/status/created_at.
-- name: GetActiveSubscriptionWithPlan :one
SELECT
  s.status::text          AS subscription_status,
  s.current_period_start  AS current_period_start,
  s.current_period_end    AS current_period_end,
  s.cancel_at_period_end  AS cancel_at_period_end,
  p.code                  AS plan_code,
  p.name                  AS plan_name
FROM subscriptions s
JOIN plans p ON p.id = s.plan_id
WHERE s.organization_id = $1
  AND s.status IN ('active', 'trialing')
ORDER BY s.created_at DESC
LIMIT 1;

-- name: ListInvoices :many
SELECT * FROM invoices
WHERE organization_id = $1
ORDER BY issued_at DESC, id DESC;

-- name: GetInvoice :one
SELECT * FROM invoices
WHERE organization_id = $1 AND id = $2;

-- name: ListInvoiceLines :many
SELECT * FROM invoice_lines
WHERE organization_id = $1 AND invoice_id = $2
ORDER BY created_at, id;
