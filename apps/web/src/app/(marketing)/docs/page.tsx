import type { Metadata } from "next";

import Link from "next/link";

export const metadata: Metadata = {
  title: "Overview",
  description:
    "AI Governance is a multi-tenant gateway and governance control plane for enterprise AI usage. Read the docs to understand the architecture, API, and deployment.",
};

const processes = [
  { name: "gateway", port: "8080", role: "Data plane: the hot-path proxy (auth, policy snapshot, rate/budget, forward, stream)" },
  { name: "gateway-v2", port: "8084", role: "Data plane v2: next-gen hot-path proxy (currently serves provider passthrough)" },
  { name: "control-api", port: "8081", role: "Control plane API: tenants, policies, credentials, audit, billing" },
  { name: "worker", port: "8082", role: "Async worker: health probes only, no jobs wired up yet" },
  { name: "migrate", port: "one-shot", role: "Applies database migrations" },
  { name: "seed", port: "one-shot", role: "Applies migrations and seeds demo data" },
  { name: "apps/web", port: "3000", role: "Next.js dashboard over the control API" },
] as const;

const why = [
  {
    title: "Route and govern",
    body: "Organizations, teams, projects, and users get scoped access to AI providers (Anthropic, OpenAI, Gemini) with API credentials.",
  },
  {
    title: "Policy in the hot path",
    body: "Authorization, rate limits, and budget checks run at request time, backed by compiled, immutable policy snapshots.",
  },
  {
    title: "Native provider compatibility",
    body: "Provider payloads pass through provider-native, including Claude Code passthrough. The gateway does not force a universal schema on the wire.",
  },
  {
    title: "Usage accounting",
    body: "Usage events carry a unit and quantity, and a database-level unique key on (organization, request_id, event_type) makes ingest idempotent. Rolling them up is not wired up yet: the async worker currently serves only health probes.",
  },
  {
    title: "Self-serve control plane",
    body: "A web app and REST API to manage tenants, policies, credentials, audit, and billing.",
  },
] as const;

export default function DocsPage() {
  return (
    <>
      <h1 className="docs-title">Governance for enterprise AI usage</h1>
      <p className="docs-lead">
        AI Governance is a multi-tenant AI gateway and governance control
        plane. Teams route AI traffic through one gateway while the control
        plane decides who can call which provider and model, enforces budgets
        and rate limits, and records audited usage.
      </p>

      <h2 className="docs-h2">Why it exists</h2>
      <ul className="docs-ul">
        {why.map((item) => (
          <li key={item.title} className="docs-li">
            <strong>{item.title}.</strong> {item.body}
          </li>
        ))}
      </ul>

      <h2 className="docs-h2">Architecture at a glance</h2>
      <p className="docs-p">
        Six Go processes plus a web app, in one monorepo. The data plane is a
        fast, stateless proxy; the control plane is the system of record.
      </p>
      <div className="docs-table-wrap">
        <table className="docs-table">
          <thead>
            <tr>
              <th>Process</th>
              <th className="docs-table-main">Role</th>
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

      <h2 className="docs-h2">Repository layout</h2>
      <pre className="docs-pre">
        <code>{`backend/    Go module: cmd/{gateway,gateway-v2,control-api,worker,migrate,seed} + internal/
apps/web    Next.js web app
packages/   Shared TS: api-client, contracts, cli
deploy/     Docker, docker-compose, Kubernetes, Terraform (AWS)
docs/       Architecture, ADRs, API, security, operations, runbooks, plan
scripts/    Dev and ops helpers`}</code>
      </pre>

      <h2 className="docs-h2">Status</h2>
      <p className="docs-p">
        M0 (repository foundation), M3 (identity, tenants, control-api auth),
        and M4 (provider layer) are shipped. The gateway hot path (M5) and the
        remaining milestones are planned. The roadmap lives in{" "}
        <code className="docs-code">docs/plan/milestones.md</code>.
      </p>
      <div className="docs-note">
        <strong>Design decisions</strong> are recorded as ADRs under{" "}
        <code className="docs-code">docs/adr</code>: monorepo and execution
        boundaries (ADR-0), hybrid architecture and contract (ADR-1),
        foundation conventions (ADR-2), persistence foundation (ADR-3), and
        control-plane auth (ADR-4).
      </div>

      <div className="docs-footer">
        Continue to{" "}
        <Link href="/docs/getting-started" className="docs-a">
          Getting started
        </Link>{" "}
        or{" "}
        <Link href="/docs/architecture" className="docs-a">
          Architecture
        </Link>
        .
      </div>
    </>
  );
}