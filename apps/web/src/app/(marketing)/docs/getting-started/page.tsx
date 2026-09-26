import type { Metadata } from "next";

import Link from "next/link";

export const metadata: Metadata = {
  title: "Getting started",
  description:
    "Run AI Governance locally: prerequisites, docker compose setup, service table, health probes, and the usage guide for dashboard, API, and gateway.",
};

const services = [
  { name: "postgres", port: "5432", role: "control-plane DB (ai_gateway)" },
  { name: "redis", port: "6379", role: "reserved; consumed once the Redis milestone lands" },
  { name: "nats", port: "4222 / 8222", role: "event bus; reserved until the eventbus milestone lands" },
  { name: "gateway", port: "8080", role: "data plane proxy" },
  { name: "gateway-v2", port: "8084", role: "data plane v2" },
  { name: "control-api", port: "8081", role: "control plane API" },
  { name: "worker", port: "8082", role: "health probes only; no jobs wired up yet" },
  { name: "web", port: "3000", role: "Next.js dashboard" },
] as const;

export default function GettingStartedPage() {
  return (
    <>
      <h1 className="docs-title">Run the full stack locally</h1>
      <p className="docs-lead">
        One docker compose file runs the database, the message bus, all five
        backend processes, and the web dashboard. No manual service wiring.
      </p>

      <h2 className="docs-h2">Prerequisites</h2>
      <ul className="docs-ul">
        <li className="docs-li">
          Go 1.25+ (backend binaries)
        </li>
        <li className="docs-li">Node.js 20+ (web app)</li>
        <li className="docs-li">Docker (local Postgres and Redis)</li>
      </ul>

      <h2 className="docs-h2">Start the stack</h2>
      <pre className="docs-pre">
        <code>{`cp deploy/compose/.env.example deploy/compose/.env   # adjust secrets if desired
make dev-up      # build + start the whole stack
make dev-seed    # optional: load demo data
make dev-down    # stop everything
make dev-logs    # tail all service logs`}</code>
      </pre>
      <p className="docs-p">
        <code className="docs-code">make dev-up</code> also runs the one-shot{" "}
        <code className="docs-code">migrate</code> service automatically: it
        applies migrations and exits, and the backend services wait for it via{" "}
        <code className="docs-code">depends_on: service_completed_successfully</code>.
        The <code className="docs-code">seed</code> service sits behind the{" "}
        <code className="docs-code">seed</code> profile, so it only runs when
        you explicitly ask (<code className="docs-code">make dev-seed</code>).
      </p>

      <h2 className="docs-h2">Services and ports</h2>
      <div className="docs-table-wrap">
        <table className="docs-table">
          <thead>
            <tr>
              <th>Service</th>
              <th className="docs-table-main">Host port</th>
              <th className="docs-table-main">Notes</th>
            </tr>
          </thead>
          <tbody>
            {services.map((service) => (
              <tr key={service.name}>
                <td>{service.name}</td>
                <td className="docs-table-main">{service.port}</td>
                <td className="docs-table-main">{service.role}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <h2 className="docs-h2">Environment</h2>
      <p className="docs-p">
        <code className="docs-code">deploy/compose/.env</code> holds the
        compose-level credentials (POSTGRES_*, REDIS_PASSWORD,
        AUTH_HMAC_SIGNING_KEY). The backend processes receive them as AIGW_*
        env vars through the compose <code className="docs-code">x-backend-env</code>{" "}
        anchor, so you normally never edit per-service env by hand.
      </p>
      <div className="docs-note">
        <strong>Health probes.</strong> Every backend process exposes{" "}
        <code className="docs-code">/livez</code> and{" "}
        <code className="docs-code">/readyz</code> (plus{" "}
        <code className="docs-code">/healthz</code>); compose uses them for
        dependency ordering and restart checks. See{" "}
        <Link href="/docs/architecture" className="docs-a">
          Architecture
        </Link>{" "}
        for the exact probe semantics.
      </div>

      <h2 className="docs-h2">Usage guide</h2>
      <ul className="docs-ul">
        <li className="docs-li">
          <strong>Dashboard:</strong>{" "}
          <code className="docs-code">http://localhost:3000</code>{" "}
          (web talks to control-api on :8081)
        </li>
        <li className="docs-li">
          <strong>Control plane API:</strong>{" "}
          <code className="docs-code">http://localhost:8081</code>{" "}
          (dev login enabled for local use; the wire contract lives in{" "}
          <code className="docs-code">backend/api/openapi/openapi.yaml</code>)
        </li>
        <li className="docs-li">
          <strong>Provider gateway:</strong>{" "}
          <code className="docs-code">http://localhost:8080</code>{" "}
          (route provider traffic through it)
        </li>
        <li className="docs-li">
          <strong>Remote envs / local runs:</strong> read{" "}
          <code className="docs-code">backend/.env.example</code> and export
          the same AIGW_* variables
        </li>
      </ul>

      <h2 className="docs-h2">Without Docker (backend only)</h2>
      <pre className="docs-pre">
        <code>{`make build       # build all Go binaries
make test        # run Go tests with race detector`}</code>
      </pre>

      <div className="docs-footer">
        Continue to{" "}
        <Link href="/docs/architecture" className="docs-a">
          Architecture
        </Link>
        {" "}or{" "}
        <Link href="/docs/deployment" className="docs-a">
          Deployment
        </Link>
        .
      </div>
    </>
  );
}