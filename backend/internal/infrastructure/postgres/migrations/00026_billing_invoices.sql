-- +goose Up
-- Invoices are local billing statements computed by this application. No payment
-- processor is integrated: there is no charge, no processor transaction id, and
-- no settlement. Status is therefore limited to local states and deliberately
-- has no 'paid' value, so a row can never imply money moved.
CREATE TABLE invoices (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id  uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    subscription_id  uuid        REFERENCES subscriptions (id) ON DELETE SET NULL,
    number           text        NOT NULL,
    status           text        NOT NULL DEFAULT 'open'
                                  CHECK (status IN ('open', 'void')),
    currency         text        NOT NULL DEFAULT 'usd',
    total_micros     bigint      NOT NULL DEFAULT 0,
    period_start     timestamptz NOT NULL,
    period_end       timestamptz NOT NULL,
    issued_at        timestamptz NOT NULL DEFAULT now(),
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, number),
    CHECK (period_end > period_start)
);
CREATE INDEX invoices_org_created_at_idx ON invoices (organization_id, created_at);

-- Invoice lines are the immutable itemisation behind an invoice total.
-- organization_id is denormalized from the parent invoice so the line table is
-- tenant-scoped in its own right, matching teams -> team_members.
CREATE TABLE invoice_lines (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid       NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    invoice_id     uuid        NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    description    text        NOT NULL,
    quantity       bigint      NOT NULL DEFAULT 1,
    unit_micros    bigint      NOT NULL DEFAULT 0,
    amount_micros  bigint      NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX invoice_lines_org_invoice_idx ON invoice_lines (organization_id, invoice_id);

-- Created after 00012, so these tables declare their own tenant guards rather
-- than joining that migration's backfill array.
-- +goose StatementBegin
DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['invoices', 'invoice_lines'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format(
            'CREATE POLICY %I ON %I USING (organization_id = app.current_org_id()) WITH CHECK (organization_id = app.current_org_id())',
            t || '_org_isolation', t);
    END LOOP;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE invoice_lines;
DROP TABLE invoices;