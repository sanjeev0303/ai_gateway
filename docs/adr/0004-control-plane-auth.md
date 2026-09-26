# ADR-4: Control-plane authentication & authorization

- **Status:** Accepted (M3)
- **Date:** 2026-09-25
- **Deciders:** Project owner, Sisyphus (implementation)

## Context

M3 ships the control plane a tenant can actually use: organizations, members,
invitations, teams, projects, service accounts and API credentials, served as
`/v1/…` HTTP endpoints by `cmd/control-api`. Before the first such endpoint is
useful, the control plane must decide *who* is calling:

- **Two distinct principals** exist. Human users sign in through the web app
  (session-style access); machine identities (CI bots, automation, the gateway
  itself) need long-lived credentials. They must not share a token format, a
  header, or a permission set.
- **Ownership semantics** (ADR-0) bind every tenant resource to an
  `organization_id`. Authorization therefore must be expressible *before* any
  repository call, so cross-tenant mistakes are rejected at the service
  boundary, not discovered at the database (which RLS enforces as a second
  line of defense, ADR-3).
- **No password store exists yet.** Real interactive login is out of scope
  until the web app lands (M9); the milestone needs a developer-facing way to
  boot an owner and prove the full journey against real Postgres.

Open questions: which credentials may the control plane accept, how do they
map to an identity the service layer can authorize, what permission model
roles grant, and how should authentication failures surface on the wire?

## Decision

### Two credential types, two headers, one actor

- **Human users authenticate with JWTs** in `Authorization: Bearer <jwt>`.
  `AuthenticateToken` submits the token to `ports.TokenVerifier` and derives
  the actor from the verified claims (`claims.Actor()`). A rejected token —
  malformed, expired, tampered, or foreign-issuer — maps to
  `identity.ErrUnauthorized`: callers cannot distinguish failure modes.
- **Machine identities authenticate with API keys** in `X-API-Key: <key>`.
  `AuthenticateAPIKey` hashes the presented key, resolves the credential by
  hash, and requires every gate to pass:
  1. the credential exists (`ErrNotFound` → 401),
  2. the credential is `active`,
  3. the credential is **not** `gateway`-scoped (gateway keys authenticate the
     data plane, never the control plane),
  4. the credential is bound to a service account,
  5. the service account exists and is `active`.
  Any failed gate yields `identity.ErrUnauthorized`.
- Both paths produce the same `domain.Actor` value, so neither the middleware
  nor service methods care which credential style arrived: a user delivers a
  user-actor, a machine delivers a service-actor.

### `requireAuth` wraps every v1 route; dev-login is the only exception

- Every `/v1/…` route runs behind `requireAuth`, which resolves the actor and
  stashes it in the request context (`withActor`/`actorFrom`). A failed
  authentication responds 401 `unauthenticated` (JSON envelope, ADR-2) and
  never reaches the handler.
- `POST /v1/auth/dev-login` is registered *only* when dev login is enabled
  (`Deps.DevLoginEnabled`); a disabled deployment simply 404s it. It signs in
  by email without a password, provisions the user and their first
  organization on demand (so a fresh install can boot an owner), and issues a
  JWT whose claims carry **no** issue/expiry timestamps (the web app and
  issuer establish session lifetime later; the wire renders them as the zero
  RFC3339 instant).

### Role-based permissions, enforced at the service boundary

- **Permissions** are coarse-grained, named capabilities
  (`org.read/manage`, `members.read/manage`, `teams.read/manage`,
  `projects.read/manage`, `providers.read/manage`,
  `credentials.read/manage`, `usage.read`, `audit.read`,
  `billing.read/manage`). Each service operation maps to exactly one
  permission.
- **Roles** map to permission sets in one table (`rolePermissions`):
  - `owner` — everything;
  - `admin` — everything except `org.manage` and `billing.manage`;
  - `auditor` — read-only across the organization;
  - `billing_admin` — reads plus `billing.manage`;
  - `member` — day-to-day reads (`org/members/teams/projects.read`,
    `usage.read`);
  - `service` — a **pseudo-role for machine identities**, granted only
    `org.read`, `usage.read`, `audit.read`, `billing.read`; never persisted,
    never stored, shaped solely by `ServiceActor`.
- Every service method starts with
  `RequirePermission(actor, perm)` → `repositories.ErrForbidden` → 403
  `permission_denied`. Authorization is decided once, in the application
  layer, before any repository interaction; handlers never re-derive it.

### API key format and lifecycle

- Raw keys are `aigw_` + 16 random bytes hex-encoded (36 chars), generated
  with a CSPRNG; the raw key is returned **exactly once** at creation. Only
  the hash is stored; a display-only `key_prefix` (`aigw_` + 7 chars)
  identifies a key in listings without exposing material.
- Scopes: `gateway` (no service account required — authenticates the data
  plane), `control` and `all` (require an active service account; the actor's
  `UserID` is the service account id, `Role` is `service`).
- Credential mutations (`api_credential.create`, `api_credential.revoke`) and
  every other tenant mutation write an audit record (M3 scope).
- All credential lookups go through the hashed-resolver port
  (`ResolveByKeyHash`), mirroring the RLS-protected
  `app.resolve_api_credential` function from ADR-3; key material never moves
  in a response or a log.

### Authentication failures are indistinguishable on the wire

One 401 `unauthenticated` covers missing credentials, malformed headers,
unknown keys, revoked credentials, missing/revoked service accounts, and
invalid tokens — no probing for valid identifiers, no enumeration. The wire
contract is the locked mapping table in the interface tests: every
identity/repository sentinel maps to one status + one machine code
(ADR-2), and unknown errors degrade to a generic 500.

## Consequences

- One `requireAuth` gate gives every current and future v1 endpoint the same
  authentication semantics; adding an endpoint cannot accidentally skip
  authn/authz.
- Roles and permissions live in one deterministic table in the domain,
  unit-testable and readable, rather than scattered `if actor.Role == …`
  checks across handlers.
- Gateway-scoped keys cannot touch the control plane by construction, and
  control-plane keys are useless on the hot path — the two planes share a
  credential table but never a credential's authority.
- The uniform 401 is blunt (a legit user cannot tell why their key failed),
  but that is deliberate: distinguishing failure modes is a fingerprinting
  oracle. Support tooling can consult audit logs instead.
- Cost: dev-login is a footgun if deployed — a live instance with
  `DevLoginEnabled` accepts any email. The route is compile-gated per
  deployment; the bootstrap wiring must keep it off outside development
  (deploy check in M10 hardening).
- PT TTL/expiry are deferred to the issuer implementation; the claims carried
  by dev-login explicitly leave them unset so no placeholder policy leaks
  into production signing.

## Alternatives considered

- **Shared token format for users and machines** — rejected: a JWT is
  session-shaped, an API key is long-lived; sharing one format forces
  awkward expiry/rotation semantics on both.
- **Access tokens in a single `Authorization` header** — rejected as
  ambiguous; two headers (`Bearer` vs `X-API-Key`) make the credential type
  explicit and parseable by middleware without inspecting payloads.
- **Per-handler authorization** — rejected: rules drift and accumulate;
  permission-gated service methods keep authorization adjacent to the
  business rule it protects.
- **Persisted per-role rows for service accounts** — rejected: the `service`
  pseudo-role is fixed by code, so the database never stores it; revoking a
  service account's access is done by status, not by role mutation.
- **Distinct 401 bodies per failure mode** — rejected: leaks which
  identifiers are valid; audit logs already capture the full picture.
- **Password-based interactive login in M3** — rejected: out of milestone
  scope; dev-login is the transitional developer path until the web app
  ships real auth (M9).