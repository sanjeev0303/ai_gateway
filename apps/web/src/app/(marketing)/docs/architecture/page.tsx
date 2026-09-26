import type { Metadata } from "next";

import Link from "next/link";

import { RequestPathDiagram } from "../request-path-diagram";

export const metadata: Metadata = {
  title: "Architecture",
  description:
    "AI Gateway architecture: four bounded execution processes, a strict dependency direction, hot-path policy snapshots, and the hybrid Go/TypeScript wire contract.",
};

const processes = [
  { name: "gateway", role: "Data plane hot path: authn, cached policy snapshot, model authz, rate/budget, forwarding, streaming" },
  { name: "control-api", role: "Control plane REST API: tenants, policies, credentials, audit, billing. The only writer of governance state" },
  { name: "worker", role: "Async worker: health probes only. Usage rollups, retention, and alerts are not implemented yet" },
  { name: "web", role: "Next.js dashboard over the control API, via @ai-governance/api-client" },
] as const;

export default function ArchitecturePage() {
  return (
    <>
      <h1 className="docs-title">Four bounded processes, one direction</h1>
      <p className="docs-lead">
        The backend is a single Go module split into bounded execution
        processes with strict dependency direction; the web app is the only
        non-Go runtime. ADR-0 settles the split; ADR-1 settles how the two
        runtimes share wire types.
      </p>

      <h2 className="docs-h2">Processes</h2>
      <div className="docs-table-wrap">
        <table className="docs-table">
          <thead>
            <tr>
              <th>Process</th>
              <th className="docs-table-main">Responsibility</th>
            </tr>
          </thead>
          <tbody>
            {processes.map((process) => (
              <tr key={process.name}>
                <td>{process.name}</td>
                <td className="docs-table-main">{process.role}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="docs-p">
        Two additional binaries exist in the repo: <code className="docs-code">gateway-v2</code>{" "}
        (the next data-plane proxy, currently serving provider passthrough on
        :8084) and the one-shot{" "}
        <code className="docs-code">migrate</code> /{" "}
        <code className="docs-code">seed</code> commands used by compose.
      </p>

      <h2 className="docs-h2">Backend structure (strict dependency direction)</h2>
      <pre className="docs-pre">
        <code>{`interfaces/  (HTTP handlers, CLI)  --->  application/  --->  domain/  --->  ports/  <---  infrastructure/`}</code>
      </pre>
      <p className="docs-p">
        Domain never imports infrastructure (pgx, Redis, cloud SDKs).{" "}
        <code className="docs-code">internal/bootstrap/</code> wires
        configuration and dependencies for each <code className="docs-code">cmd/</code>{" "}
        binary, so every command stays thin: read config, assemble, run.{" "}
        <code className="docs-code">scripts/check-boundaries.sh</code> enforces
        the direction in CI using <code className="docs-code">go list</code>.
      </p>

      <h2 className="docs-h2">The hot path</h2>
      <pre className="docs-pre">
        <code>{`authn  ->  resolve cached, compiled policy snapshot  ->  authorize model/credential
       ->  enforce rate + budget  ->  forward  ->  stream`}</code>
      </pre>
      <figure className="docs-figure">
        <RequestPathDiagram />
      </figure>
      <ul className="docs-ul">
        <li className="docs-li">
          <strong>Gateway never queries Postgres on the hot path.</strong> It
          consumes immutable policy snapshots (org to team to project to
          user/service-account to credential) served from an in-memory cache,
          Redis-backed at startup, invalidated by the control API via event.
        </li>
        <li className="docs-li">
          <strong>Usage accounting is specified, not yet wired.</strong> The{" "}
          <code className="docs-code">usage_events</code> table carries a unit,
          a quantity, and a unique key on{" "}
          <code className="docs-code">
            (organization_id, request_id, event_type)
          </code>{" "}
          that makes ingest idempotent. Neither data plane writes to it yet, and
          the worker runs no jobs, so nothing is recorded at runtime today.
        </li>
        <li className="docs-li">
          <strong>Provider-native payloads on the wire,</strong> including
          Claude Code passthrough. Normalization happens after the request, in
          the worker/application layer, never by rewriting the request.
        </li>
      </ul>

      <h2 className="docs-h2">Communication boundaries</h2>
      <ul className="docs-ul">
        <li className="docs-li">
          <strong>Control API is the single writer</strong> of governance state
          in Postgres.
        </li>
        <li className="docs-li">
          <strong>Eventbus behind a port:</strong> an in-process/Postgres-backed
          queue first; a NATS/queue-backed implementation sits behind the same{" "}
          <code className="docs-code">ports/eventbus</code> interface and is
          introduced only when volume justifies it.
        </li>
        <li className="docs-li">
          <strong>Infrastructure:</strong> Postgres (primary store) and Redis
          (policy snapshot cache, rate-limit counters) locally via Docker
          Compose, managed equivalents in prod. Stdlib <code className="docs-code">net/http</code>,{" "}
          <code className="docs-code">log/slog</code>, and OpenTelemetry with a
          Prometheus scrape endpoint per process.
        </li>
      </ul>

      <h2 className="docs-h2">The wire contract (hybrid architecture)</h2>
      <p className="docs-p">
        Go implements every backend process; TypeScript/Next.js implements the
        web app. The two runtimes share one contract without drift (ADR-1):
      </p>
      <ul className="docs-ul">
        <li className="docs-li">
          <strong>One source of truth:</strong>{" "}
          <code className="docs-code">backend/api/openapi/openapi.yaml</code>{" "}
          (OpenAPI 3.0) defines every wire type, growing milestone by milestone.
        </li>
        <li className="docs-li">
          <strong>TS types are generated, never hand-written:</strong>{" "}
          <code className="docs-code">openapi-typescript</code> emits raw
          TypeScript into <code className="docs-code">packages/contracts/src/generated.ts</code>.
        </li>
        <li className="docs-li">
          <strong>Go implements endpoints by hand with the stdlib.</strong> No
          code generator on the backend, preserving the stdlib-first rule and
          keeping handlers explicit.
        </li>
        <li className="docs-li">
          <strong>The web app never imports contracts directly.</strong> All
          network access goes through <code className="docs-code">@ai-governance/api-client</code>,
          which owns parsing, the single permitted JSON-boundary cast, and error
          surfacing.
        </li>
        <li className="docs-li">
          <strong>Errors are contractual:</strong> the{" "}
          <code className="docs-code">ErrorEnvelope</code> schema bridges Go's
          kind-based error taxonomy and the TS client (see below).
        </li>
      </ul>

      <h2 className="docs-h2">Errors, health, config</h2>
      <ul className="docs-ul">
        <li className="docs-li">
          <strong>One error taxonomy</strong> in{" "}
          <code className="docs-code">platform/errors</code>: nine kinds each
          mapped to a stable machine code (
          <code className="docs-code">invalid_argument</code>,{" "}
          <code className="docs-code">unauthenticated</code>,{" "}
          <code className="docs-code">permission_denied</code>,{" "}
          <code className="docs-code">not_found</code>,{" "}
          <code className="docs-code">conflict</code>,{" "}
          <code className="docs-code">rate_limited</code>,{" "}
          <code className="docs-code">timeout</code>,{" "}
          <code className="docs-code">unavailable</code>,{" "}
          <code className="docs-code">internal</code>). Every API endpoint
          returns the envelope on failure, nothing else.
        </li>
        <li className="docs-li">
          <strong>Health:</strong> every process serves{" "}
          <code className="docs-code">/livez</code> (liveness),{" "}
          <code className="docs-code">/readyz</code> (readiness with dependency
          checks), and <code className="docs-code">/healthz</code> (aggregate
          with per-check detail). Body is{" "}
          <code className="docs-code">{"{status, checks}"}</code>, 200 when ok,
          503 when degraded.
        </li>
        <li className="docs-li">
          <strong>Config:</strong> environment-only,{" "}
          <code className="docs-code">AIGW_</code> prefix, validated at startup
          (fail fast). <code className="docs-code">.env</code> is local-dev only
          and never committed; <code className="docs-code">backend/.env.example</code>{" "}
          documents every variable.
        </li>
        <li className="docs-li">
          <strong>HTTP plumbing</strong> in{" "}
          <code className="docs-code">platform/httpx</code>: request ID
          (<code className="docs-code">X-Request-ID</code>), panic recovery
          (single 500 envelope, no stack leak), request timeout (504 envelope),
          per-request structured logging.
        </li>
      </ul>

      <h2 className="docs-h2">Persistence and authorization</h2>
      <ul className="docs-ul">
        <li className="docs-li">
          <strong>Postgres 16</strong> via pgx/v5 + pgxpool; sqlc-generated
          queries; goose v3 with forward-only embedded migrations, applied by
          the one-shot migrate service. RLS is the second line of defense:
          authorization is decided at the service boundary before any
          repository call, and <code className="docs-code">organization_id</code>{" "}
          sits on every tenant table (ADR-3).
        </li>
        <li className="docs-li">
          <strong>Two credential types, two headers, one actor (ADR-4):</strong>{" "}
          humans authenticate with JWTs in{" "}
          <code className="docs-code">Authorization: Bearer &lt;jwt&gt;</code>;
          machine identities use API keys in{" "}
          <code className="docs-code">X-API-Key: &lt;key&gt;</code>. Both resolve
          to the same <code className="docs-code">domain.Actor</code>, and every{" "}
          <code className="docs-code">/v1/&hellip;</code> route runs behind{" "}
          <code className="docs-code">requireAuth</code>. See{" "}
          <Link href="/docs/api" className="docs-a">
            Control plane API
          </Link>
          .
        </li>
      </ul>
      <div className="docs-note">
        <strong>Design decisions</strong> are written up as ADRs in{" "}
        <code className="docs-code">docs/adr/</code> (0000 through 0004). Each
        records the context, the decision, consequences, and alternatives
        considered.
      </div>

      <div className="docs-footer">
        Continue to{" "}
        <Link href="/docs/api" className="docs-a">
          Control plane API
        </Link>{" "}
        or{" "}
        <Link href="/docs/deployment" className="docs-a">
          Deployment
        </Link>
        .
      </div>
    </>
  );
}