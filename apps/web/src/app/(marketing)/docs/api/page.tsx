import type { Metadata } from "next";

import Link from "next/link";

export const metadata: Metadata = {
  title: "Control plane API",
  description:
    "The control plane API: authenticated /v1 endpoints for members, invitations, projects, teams, service accounts, credentials, and provider accounts, plus the error contract.",
};

const endpointGroups = [
  {
    group: "Auth",
    endpoints: [
      { method: "POST", path: "/v1/auth/dev-login", note: "Registers only when dev login is enabled; 404 otherwise" },
      { method: "POST", path: "/v1/auth/logout", note: "Registers only when the session cookie is configured" },
    ],
  },
  {
    group: "Members",
    endpoints: [
      { method: "GET", path: "/v1/members", note: "" },
      { method: "PATCH", path: "/v1/members/{userID}/role", note: "" },
      { method: "DELETE", path: "/v1/members/{userID}", note: "" },
    ],
  },
  {
    group: "Invitations",
    endpoints: [
      { method: "POST", path: "/v1/invitations", note: "Invite a member" },
      { method: "POST", path: "/v1/invitations/accept", note: "Accept an invitation" },
      { method: "GET", path: "/v1/invitations", note: "" },
      { method: "DELETE", path: "/v1/invitations/{id}", note: "Revoke" },
    ],
  },
  {
    group: "Projects",
    endpoints: [
      { method: "POST", path: "/v1/projects", note: "" },
      { method: "GET", path: "/v1/projects", note: "" },
      { method: "GET", path: "/v1/projects/{id}", note: "" },
      { method: "POST", path: "/v1/projects/{id}/members", note: "" },
      { method: "GET", path: "/v1/projects/{id}/members", note: "" },
      { method: "DELETE", path: "/v1/projects/{id}/members/{userID}", note: "" },
    ],
  },
  {
    group: "Teams",
    endpoints: [
      { method: "POST", path: "/v1/teams", note: "" },
      { method: "GET", path: "/v1/teams", note: "" },
      { method: "GET", path: "/v1/teams/{id}", note: "" },
      { method: "POST", path: "/v1/teams/{id}/members", note: "" },
      { method: "GET", path: "/v1/teams/{id}/members", note: "" },
      { method: "DELETE", path: "/v1/teams/{id}/members/{userID}", note: "" },
    ],
  },
  {
    group: "Service accounts and credentials",
    endpoints: [
      { method: "POST", path: "/v1/service-accounts", note: "" },
      { method: "GET", path: "/v1/service-accounts", note: "" },
      { method: "DELETE", path: "/v1/service-accounts/{id}", note: "Revoke" },
      { method: "POST", path: "/v1/credentials", note: "Create an API key" },
      { method: "GET", path: "/v1/credentials", note: "" },
      { method: "DELETE", path: "/v1/credentials/{id}", note: "Revoke" },
      { method: "POST", path: "/v1/gateway-credentials", note: "Gateway-scoped key" },
      { method: "GET", path: "/v1/gateway-credentials", note: "" },
      { method: "GET", path: "/v1/gateway-credentials/{id}", note: "" },
      { method: "DELETE", path: "/v1/gateway-credentials/{id}", note: "Revoke" },
      { method: "POST", path: "/v1/gateway-credentials/{id}/rotate", note: "" },
      { method: "POST", path: "/v1/gateway-credentials/{id}/expire", note: "" },
      { method: "POST", path: "/v1/gateway-credentials/{id}/restore", note: "" },
    ],
  },
  {
    group: "Provider accounts",
    endpoints: [
      { method: "POST", path: "/v1/provider-accounts", note: "" },
      { method: "GET", path: "/v1/provider-accounts", note: "" },
      { method: "GET", path: "/v1/provider-accounts/{id}", note: "" },
      { method: "POST", path: "/v1/provider-accounts/{id}/credentials", note: "" },
      { method: "GET", path: "/v1/provider-credentials", note: "" },
      { method: "GET", path: "/v1/provider-credentials/{id}", note: "" },
      { method: "DELETE", path: "/v1/provider-credentials/{id}", note: "Revoke" },
      { method: "POST", path: "/v1/provider-credentials/{id}/rotate", note: "" },
      { method: "GET", path: "/v1/models", note: "Provider model catalog" },
    ],
  },
] as const;

export default function ApiPage() {
  return (
    <>
      <h1 className="docs-title">The v1 control plane API</h1>
      <p className="docs-lead">
        The control plane is served by <code className="docs-code">control-api</code>{" "}
        on <code className="docs-code">:8081</code>. Every{" "}
        <code className="docs-code">/v1/&hellip;</code> route runs behind{" "}
        <code className="docs-code">requireAuth</code>; dev-login is the only
        exception, and a disabled deployment simply 404s it.
      </p>

      <h2 className="docs-h2">Authentication</h2>
      <p className="docs-p">
        Two credential types, two headers, one actor:
      </p>
      <ul className="docs-ul">
        <li className="docs-li">
          <strong>Humans:</strong> JWTs in{" "}
          <code className="docs-code">Authorization: Bearer &lt;jwt&gt;</code>.
          The token is verified and the actor derived from the claims.
        </li>
        <li className="docs-li">
          <strong>Machine identities:</strong> API keys in{" "}
          <code className="docs-code">X-API-Key: &lt;key&gt;</code>. The
          presented key is hashed and resolved by hash; every gate must pass
          (exists, active, not gateway-scoped, bound to an active service
          account).
        </li>
      </ul>
      <p className="docs-p">
        Both paths produce the same <code className="docs-code">domain.Actor</code>,
        so handlers never care which credential style arrived. A missing,
        malformed, expired, or unknown credential is indistinguishable on the
        wire: one <code className="docs-code">401 unauthenticated</code>, by
        design, to avoid fingerprinting (audit logs hold the detail).
      </p>
      <div className="docs-note">
        <strong>API keys.</strong> Raw keys are{" "}
        <code className="docs-code">aigw_</code> + 16 random bytes hex-encoded
        (36 chars), generated with a CSPRNG and returned exactly once at
        creation. Only the hash is stored; a display-only{" "}
        <code className="docs-code">key_prefix</code> identifies a key in
        listings. Key material never appears in responses or logs.
      </div>

      <h2 className="docs-h2">Endpoints</h2>
      <p className="docs-p">
        Endpoint groups below are registered as their dependencies are wired
        in: the identity surface when the identity service is present, the
        provider surface when the provider service joins it. The single source
        of truth for every wire type and error is{" "}
        <code className="docs-code">backend/api/openapi/openapi.yaml</code>,
        which generates <code className="docs-code">packages/contracts</code>.
      </p>
      {endpointGroups.map((group) => (
        <section key={group.group} className="docs-api-group">
          <h3 className="docs-h3">{group.group}</h3>
          <div className="docs-table-wrap">
            <table className="docs-table">
              <thead>
                <tr>
                  <th>Method</th>
                  <th>Path</th>
                  {group.endpoints.some((e) => e.note) && (
                    <th className="docs-table-main" />
                  )}
                </tr>
              </thead>
              <tbody>
                {group.endpoints.map((endpoint) => (
                  <tr key={`${endpoint.method}-${endpoint.path}`}>
                    <td>{endpoint.method}</td>
                    <td>
                      <code className="docs-code">{endpoint.path}</code>
                    </td>
                    {group.endpoints.some((e) => e.note) && (
                      <td className="docs-table-main">{endpoint.note}</td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      ))}

      <h2 className="docs-h2">Authorization</h2>
      <p className="docs-p">
        Every service operation maps to exactly one named permission
        (<code className="docs-code">org.read/manage</code>,{" "}
        <code className="docs-code">members.read/manage</code>,{" "}
        <code className="docs-code">teams.read/manage</code>,{" "}
        <code className="docs-code">projects.read/manage</code>,{" "}
        <code className="docs-code">providers.read/manage</code>,{" "}
        <code className="docs-code">credentials.read/manage</code>,{" "}
        <code className="docs-code">usage.read</code>,{" "}
        <code className="docs-code">audit.read</code>,{" "}
        <code className="docs-code">billing.read/manage</code>). Roles map to
        permission sets: <code className="docs-code">owner</code> (everything),{" "}
        <code className="docs-code">admin</code>,{" "}
        <code className="docs-code">auditor</code>,{" "}
        <code className="docs-code">billing_admin</code>,{" "}
        <code className="docs-code">member</code>, and the{" "}
        <code className="docs-code">service</code> pseudo-role for machine
        identities (read-only). Missing permission is a{" "}
        <code className="docs-code">403 permission_denied</code>.
      </p>

      <h2 className="docs-h2">Errors</h2>
      <p className="docs-p">
        Every endpoint returns the{" "}
        <code className="docs-code">ErrorEnvelope</code> on failure: a stable
        machine code, a safe-for-users message, and a request ID. Codes:
        <code className="docs-code">invalid_argument</code>,{" "}
        <code className="docs-code">unauthenticated</code>,{" "}
        <code className="docs-code">permission_denied</code>,{" "}
        <code className="docs-code">not_found</code>,{" "}
        <code className="docs-code">conflict</code>,{" "}
        <code className="docs-code">rate_limited</code>,{" "}
        <code className="docs-code">timeout</code>,{" "}
        <code className="docs-code">unavailable</code>,{" "}
        <code className="docs-code">internal</code>. Unknown errors degrade to{" "}
        <code className="docs-code">internal</code>. Secrets, stack traces, and
        provider internals never appear in a response.
      </p>

      <h2 className="docs-h2">Health and system</h2>
      <ul className="docs-ul">
        <li className="docs-li">
          <code className="docs-code">GET /livez</code>,{" "}
          <code className="docs-code">GET /readyz</code>,{" "}
          <code className="docs-code">GET /healthz</code>:{" "}
          <code className="docs-code">{"{status, checks}"}</code>, 200 when ok,
          503 when degraded.
        </li>
        <li className="docs-li">
          Every response carries a request ID ({" "}
          <code className="docs-code">X-Request-ID</code>, generated when
          absent, echoed in logs).
        </li>
      </ul>
      <div className="docs-note">
        <strong>Data plane.</strong> The provider gateway v2 ({" "}
        <code className="docs-code">gateway-v2</code>, :8084) serves the
        provider-native surface:{" "}
        <code className="docs-code">POST /v1/messages</code>,{" "}
        <code className="docs-code">POST /v1/messages/count_tokens</code>,{" "}
        <code className="docs-code">GET /v1/models</code>, plus{" "}
        <code className="docs-code">/v1/gateway/stats</code> and{" "}
        <code className="docs-code">/v1/gateway/health</code>.
      </div>

      <div className="docs-footer">
        Continue to{" "}
        <Link href="/docs/deployment" className="docs-a">
          Deployment
        </Link>
        {" "}or back to{" "}
        <Link href="/docs/architecture" className="docs-a">
          Architecture
        </Link>
        .
      </div>
    </>
  );
}