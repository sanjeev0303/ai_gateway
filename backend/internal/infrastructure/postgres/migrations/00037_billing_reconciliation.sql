-- +goose Up
-- 1. Extend invoices status check to include full lifecycle states
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_status_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_status_check
    CHECK (status IN ('open', 'void', 'paid', 'draft', 'uncollectible'));

-- 2. Prevent duplicate invoice creation for the same payment transaction
CREATE UNIQUE INDEX IF NOT EXISTS invoices_org_payment_id_unique
    ON invoices (organization_id, payment_id)
    WHERE payment_id IS NOT NULL AND payment_id != '';

-- 3. Extend plans table with trusted server-side pricing definitions
ALTER TABLE plans ADD COLUMN IF NOT EXISTS price_micros bigint NOT NULL DEFAULT 0;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS currency text NOT NULL DEFAULT 'usd';
ALTER TABLE plans ADD COLUMN IF NOT EXISTS billing_interval text NOT NULL DEFAULT 'month';

-- Seed authoritative base pricing on existing default plans
UPDATE plans SET price_micros = 0, currency = 'usd', billing_interval = 'month' WHERE code = 'free';
UPDATE plans SET price_micros = 49000000, currency = 'usd', billing_interval = 'month' WHERE code = 'pro';
UPDATE plans SET price_micros = 499000000, currency = 'usd', billing_interval = 'month' WHERE code = 'enterprise';

-- 4. Create billing_checkouts table for tracking trusted server-side checkout state
CREATE TABLE IF NOT EXISTS billing_checkouts (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id     uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    checkout_session_id text NOT NULL,
    provider            text NOT NULL,
    plan_code           text NOT NULL,
    plan_id             uuid REFERENCES plans(id) ON DELETE SET NULL,
    status              text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'expired', 'cancelled')),
    amount_micros       bigint NOT NULL DEFAULT 0,
    currency            text NOT NULL DEFAULT 'usd',
    customer_email      text NOT NULL DEFAULT '',
    customer_id         text NOT NULL DEFAULT '',
    order_id            text NOT NULL DEFAULT '',
    payment_id          text NOT NULL DEFAULT '',
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    expires_at          timestamptz NOT NULL,
    completed_at        timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, checkout_session_id)
);

CREATE INDEX IF NOT EXISTS billing_checkouts_org_idx ON billing_checkouts (organization_id, status);
CREATE INDEX IF NOT EXISTS billing_checkouts_session_idx ON billing_checkouts (checkout_session_id);
CREATE INDEX IF NOT EXISTS billing_checkouts_order_idx ON billing_checkouts (organization_id, order_id) WHERE order_id != '';

DROP TRIGGER IF EXISTS billing_checkouts_set_updated_at ON billing_checkouts;
CREATE TRIGGER billing_checkouts_set_updated_at BEFORE UPDATE ON billing_checkouts FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE billing_checkouts ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing_checkouts FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS billing_checkouts_org_isolation ON billing_checkouts;
CREATE POLICY billing_checkouts_org_isolation ON billing_checkouts
    USING (organization_id = app.current_org_id())
    WITH CHECK (organization_id = app.current_org_id());

-- +goose Down
DROP POLICY IF EXISTS billing_checkouts_org_isolation ON billing_checkouts;
DROP TABLE IF EXISTS billing_checkouts;

ALTER TABLE plans DROP COLUMN IF EXISTS billing_interval;
ALTER TABLE plans DROP COLUMN IF EXISTS currency;
ALTER TABLE plans DROP COLUMN IF EXISTS price_micros;

DROP INDEX IF EXISTS invoices_org_payment_id_unique;

ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_status_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_status_check
    CHECK (status IN ('open', 'void', 'paid'));
