-- +goose Up
-- resolve_api_credential must return a SET, not a single composite. A
-- LANGUAGE sql function declared RETURNS api_credentials that matches zero
-- rows yields one NULL composite row, which surfaces as
--   cannot scan NULL into *string
-- instead of pgx.ErrNoRows. With RETURNS SETOF api_credentials a missing
-- key_hash stays a zero-row result, so the repository maps it to
-- ErrNotFound. 00012 is left untouched (forward-only); this migration
-- redefines the function for existing databases.
DROP FUNCTION app.resolve_api_credential(text);
-- +goose StatementBegin
CREATE FUNCTION app.resolve_api_credential(p_hash text) RETURNS SETOF api_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT * FROM api_credentials WHERE key_hash = p_hash LIMIT 1;
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION app.resolve_api_credential(text);
-- +goose StatementBegin
CREATE FUNCTION app.resolve_api_credential(p_hash text) RETURNS api_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT * FROM api_credentials WHERE key_hash = p_hash LIMIT 1;
$$;
-- +goose StatementEnd