import type { Metadata } from "next";

import Link from "next/link";

export const metadata: Metadata = {
  title: "Deployment",
  description:
    "Deploy AI Gateway: docker compose as the supported local and self-host path, environment configuration, secrets, and the deploy/ directory layout.",
};

const envVars = [
  { name: "POSTGRES_USER / POSTGRES_PASSWORD / POSTGRES_DB", role: "Control-plane database credentials and database name (ai_gateway)" },
  { name: "REDIS_PASSWORD", role: "Redis password; provisioned and consumed once the Redis milestone lands" },
  { name: "AUTH_HMAC_SIGNING_KEY", role: "Session signing key for postgres + control-api. Must be at least 32 bytes; generate with openssl rand -hex 32" },
] as const;

const dirs = [
  { path: "deploy/compose/", status: "the working docker compose setup: docker-compose.yml + .env.example" },
  { path: "deploy/docker/", status: "scaffolding placeholder for image build config" },
  { path: "deploy/kubernetes/", status: "scaffolding placeholder for Kubernetes manifests" },
  { path: "deploy/terraform/aws/", status: "scaffolding placeholder for managed AWS (EKS/RDS/ElastiCache)" },
] as const;

export default function DeploymentPage() {
  return (
    <>
      <h1 className="docs-title">Compose-first deployment</h1>
      <p className="docs-lead">
        Docker compose in <code className="docs-code">deploy/compose/</code> is
        the supported path today for local development and self-hosting. The
        rest of <code className="docs-code">deploy/</code> is scaffolding for
        the managed-infrastructure milestone.
      </p>

      <h2 className="docs-h2">Start with the compose file</h2>
      <pre className="docs-pre">
        <code>{`cp deploy/compose/.env.example deploy/compose/.env   # set real secrets
make dev-up`}</code>
      </pre>
      <p className="docs-p">
        <code className="docs-code">make dev-up</code> builds and starts the
        database, Redis, NATS, migrations, all six backend processes, and the
        web dashboard. Backend services receive their{" "}
        <code className="docs-code">AIGW_*</code> env vars through the compose{" "}
        <code className="docs-code">x-backend-env</code> anchor, so you normally
        never edit per-service environment by hand.
      </p>

      <h2 className="docs-h2">Environment and secrets</h2>
      <p className="docs-p">
        <code className="docs-code">deploy/compose/.env</code> holds the
        compose-level credentials:
      </p>
      <div className="docs-table-wrap">
        <table className="docs-table">
          <thead>
            <tr>
              <th>Variable</th>
              <th className="docs-table-main">Purpose</th>
            </tr>
          </thead>
          <tbody>
            {envVars.map((variable) => (
              <tr key={variable.name}>
                <td>
                  <code className="docs-code">{variable.name}</code>
                </td>
                <td className="docs-table-main">{variable.role}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="docs-note">
        <strong>Never commit a real .env.</strong> Values in the example file
        are obviously-fake placeholders. Generate a real signing key with{" "}
        <code className="docs-code">openssl rand -hex 32</code>; the key is
        required by postgres and control-api and must be at least 32 bytes.
      </div>

      <h2 className="docs-h2">Migrations and seeds</h2>
      <ul className="docs-ul">
        <li className="docs-li">
          The one-shot <code className="docs-code">migrate</code> service
          applies forward-only goose migrations (embedded in the binary) and
          exits 0 on success.
        </li>
        <li className="docs-li">
          Backend services wait for it via{" "}
          <code className="docs-code">depends_on: service_completed_successfully</code>,
          so a freshly-provisioned stack never races its own schema.
        </li>
        <li className="docs-li">
          <code className="docs-code">seed</code> sits behind the{" "}
          <code className="docs-code">seed</code> profile: it only runs when you
          explicitly ask (<code className="docs-code">make dev-seed</code>).
        </li>
      </ul>

      <h2 className="docs-h2">Backend environment reference</h2>
      <p className="docs-p">
        Every backend variable is prefixed{" "}
        <code className="docs-code">AIGW_</code> and documented with
        placeholders in <code className="docs-code">backend/.env.example</code>:
      </p>
      <ul className="docs-ul">
        <li className="docs-li">
          <strong>Server:</strong> <code className="docs-code">AIGW_SERVER_ADDR</code>,{" "}
          read/timeout/shutdown timeouts.
        </li>
        <li className="docs-li">
          <strong>Database:</strong> <code className="docs-code">AIGW_DATABASE_URL</code>{" "}
          (Postgres), pool sizes, connect timeout;{" "}
          <code className="docs-code">AIGW_DATABASE_ENABLED</code>.
        </li>
        <li className="docs-li">
          <strong>Redis:</strong> <code className="docs-code">AIGW_REDIS_ADDR</code>,{" "}
          password, DB; disabled until the Redis milestone lands.
        </li>
        <li className="docs-li">
          <strong>Event bus:</strong> <code className="docs-code">AIGW_EVENTBUS_KIND</code>{" "}
          (none/postgres/kafka/nats) and URL; disabled until the eventbus
          milestone lands.
        </li>
        <li className="docs-li">
          <strong>Providers:</strong> request timeout, default max retries.
        </li>
        <li className="docs-li">
          <strong>Secrets:</strong> <code className="docs-code">AIGW_SECRETS_PROVIDER</code>{" "}
          (env/vault/aws) and file location. The env provider stores values in
          a local encrypted file; never in the process config.
        </li>
        <li className="docs-li">
          <strong>Auth:</strong> JWT issuer/TTL, session TTL and cookie, HMAC
          signing key (plus rotation via previous keys), CSRF toggle, and{" "}
          <code className="docs-code">AIGW_AUTH_DEV_LOGIN_ENABLED</code>{" "}
          (dev-only; keep it false outside development).
        </li>
        <li className="docs-li">
          <strong>Billing:</strong> disabled until the billing milestone lands.
        </li>
        <li className="docs-li">
          <strong>Observability:</strong> log level/format, OTLP endpoint
          (empty = tracing disabled), metrics toggle.
        </li>
        <li className="docs-li">
          <strong>Security:</strong> CORS allowed origins,{" "}
          <code className="docs-code">AIGW_SECURITY_TRUST_PROXY</code>, request
          ID header.
        </li>
      </ul>
      <p className="docs-p">
        Configuration is validated at startup: a process refuses to boot on
        invalid or missing required variables (fail fast).
      </p>

      <h2 className="docs-h2">deploy/ directory layout</h2>
      <div className="docs-table-wrap">
        <table className="docs-table">
          <thead>
            <tr>
              <th>Path</th>
              <th className="docs-table-main">Status</th>
            </tr>
          </thead>
          <tbody>
            {dirs.map((dir) => (
              <tr key={dir.path}>
                <td>
                  <code className="docs-code">{dir.path}</code>
                </td>
                <td className="docs-table-main">{dir.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <h2 className="docs-h2">Health and observability</h2>
      <ul className="docs-ul">
        <li className="docs-li">
          Every backend process exposes{" "}
          <code className="docs-code">/livez</code>,{" "}
          <code className="docs-code">/readyz</code>, and{" "}
          <code className="docs-code">/healthz</code>; compose uses them for
          dependency ordering and restart checks.
        </li>
        <li className="docs-li">
          Per-request structured logs with a request ID, and a Prometheus
          scrape endpoint per process; tracing is ready behind the OTLP
          endpoint.
        </li>
      </ul>

      <div className="docs-footer">
        Back to{" "}
        <Link href="/docs/getting-started" className="docs-a">
          Getting started
        </Link>{" "}
        or{" "}
        <Link href="/docs" className="docs-a">
          Overview
        </Link>
        .
      </div>
    </>
  );
}