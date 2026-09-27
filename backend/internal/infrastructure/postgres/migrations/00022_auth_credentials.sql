-- +goose Up
-- Password credentials and refresh-token rotation for the control plane.
--
-- Password auth: users gained a credential. The column is NULLABLE on purpose:
-- users provisioned through the dev-login path (and seeded demo data) have no
-- password and must never be able to password-login. Sign-in requires a
-- non-NULL password_hash, so a NULL credential is simply "not a password
-- account" rather than a special case every read path has to handle.
--
-- The stored form is a self-describing string produced by the password hasher
-- (algorithm, iteration count, salt, derived key) so parameters can be raised
-- later without a schema change: old hashes stay verifiable and get upgraded
-- on the next successful sign-in.

ALTER TABLE users ADD COLUMN password_hash text;

-- Refresh tokens: opaque, high-entropy secrets handed to the browser in an
-- HttpOnly cookie. Only the SHA-256 of each token is stored, so a database
-- leak does not yield usable refresh credentials.
--
-- There is deliberately NO organization_id and NO row-level security here,
-- unlike the tenant tables. Refresh happens in a *pre-authentication* context:
-- the caller has presented only the refresh token, so the app.org_id GUC is
-- not set and an org-isolation policy would fail closed on every legitimate
-- refresh (the same reason api_credentials needs a SECURITY DEFINER lookup).
-- Rows are reached only by user_id or by token_hash, and both values are
-- derived server-side from the presented token -- never from client input --
-- so cross-tenant access has no path in. This mirrors app.resolve_api_credential.
CREATE TABLE auth_refresh_tokens (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash   text        NOT NULL UNIQUE,
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz
);

-- Lookup by presented token.
CREATE INDEX auth_refresh_tokens_user_id_idx ON auth_refresh_tokens (user_id);
-- Supports periodic pruning of expired/revoked rows. No cleanup job runs yet;
-- rows accumulate until one is added.
CREATE INDEX auth_refresh_tokens_expires_at_idx ON auth_refresh_tokens (expires_at);

-- +goose Down
DROP TABLE auth_refresh_tokens;
ALTER TABLE users DROP COLUMN password_hash;
