-- name: CreateInvitation :one
INSERT INTO invitations (organization_id, email, role, token_hash, expires_at, invited_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ResolveInvitationByToken :one
SELECT * FROM app.resolve_invitation_by_token($1);

-- name: GetInvitation :one
SELECT * FROM invitations
WHERE organization_id = $1 AND id = $2;

-- name: ListInvitations :many
SELECT * FROM invitations
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: MarkInvitationAccepted :one
UPDATE invitations
SET status = 'accepted', accepted_by = $2, accepted_at = now()
WHERE organization_id = $1 AND id = $3
RETURNING *;

-- name: RevokeInvitation :one
UPDATE invitations
SET status = 'revoked'
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: PurgeExpiredInvitations :execrows
DELETE FROM invitations
WHERE organization_id = $1 AND (expires_at < $2 OR (status IN ('accepted', 'revoked') AND created_at < $3));