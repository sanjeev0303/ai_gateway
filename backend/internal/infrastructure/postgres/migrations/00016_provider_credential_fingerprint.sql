-- +goose Up
-- M4: provider credential fingerprinting.
--
-- fingerprint is a sha256:<hex> digest of the raw secret material; tail is the
-- last four characters of the raw secret so the UI can render
--   sk-ant-••••••91A7
-- without ever exposing the secret itself. Application code always computes
-- both values at create time, so fingerprint begins NULL only for rows that
-- predate this migration (they are not rotatable under the uniqueness rule
-- until re-created).
--
-- The partial unique index guarantees an organization cannot have two ACTIVE
-- credentials carrying the same secret: rotation cannot silently re-register
-- a previously used key. NULL fingerprints never match the index, and revoked
-- credentials release their fingerprint, so a rotated key can come back later
-- on purpose (explicitly, not by accident).

ALTER TABLE provider_credentials
    ADD COLUMN fingerprint text;

ALTER TABLE provider_credentials
    ADD COLUMN tail text NOT NULL DEFAULT '';

CREATE UNIQUE INDEX provider_credentials_org_fingerprint_active_idx
    ON provider_credentials (organization_id, fingerprint)
    WHERE status = 'active' AND fingerprint IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS provider_credentials_org_fingerprint_active_idx;

ALTER TABLE provider_credentials DROP COLUMN tail;
ALTER TABLE provider_credentials DROP COLUMN fingerprint;