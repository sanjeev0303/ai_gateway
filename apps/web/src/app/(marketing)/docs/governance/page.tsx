import type { Metadata } from "next";

import Link from "next/link";

export const metadata: Metadata = {
  title: "Governance",
  description:
    "Reference for the policy primitives the gateway enforces on every request: rule types, enforcement modes, rate limit scopes, budget controls, model authorization, and policy snapshots.",
};

const ruleTypes = [
  { type: "rate_limit", applies: "Requests per window, per scope." },
  { type: "token_budget", applies: "Token spend per window, per scope." },
  { type: "budget", applies: "Cost spend per window, per scope." },
  { type: "blocklist", applies: "Named models are denied." },
  { type: "allowlist", applies: "Only named models are permitted." },
  { type: "content_filter", applies: "Request or response content." },
] as const;

const modes = [
  {
    mode: "active",
    behavior: "Every enabled rule is evaluated and enforced.",
  },
  {
    mode: "permissive",
    behavior:
      "Rules are evaluated and logged, but a failure does not block traffic.",
  },
  {
    mode: "disabled",
    behavior: "The policy is not applied to live traffic.",
  },
] as const;

const scopes = [
  { scope: "organization", meaning: "One bucket shared by every caller in the org." },
  { scope: "project", meaning: "One bucket per project." },
  { scope: "credential", meaning: "One bucket per API credential." },
  { scope: "user", meaning: "One bucket per authenticated user or service account." },
] as const;

const defaults = [
  ["Request timeout", "30s"],
  ["Upstream timeout", "5m"],
  ["Idle timeout", "90s"],
  ["Response header timeout", "30s"],
  ["Max idle connections", "1000"],
  ["Max idle connections per host", "100"],
  ["Max connections per host", "200"],
  ["Max queued requests", "10000"],
  ["HTTP/2", "enabled"],
  ["Compression", "disabled (byte-exact proxy)"],
  ["Default rate limit", "1000 per 1 minute"],
  ["Default budget", "1000.00 per 1 hour"],
] as const;

export default function GovernancePage() {
  return (
    <>
      <h1 className="docs-title">What the gateway enforces</h1>
      <p className="docs-lead">
        A policy is compiled into an immutable snapshot, cached, and consulted on
        every request. This page lists the exact rule vocabulary, scopes, result
        fields, and defaults, with the type that defines each one.
      </p>

      <h2 className="docs-h2">Rule types</h2>
      <p className="docs-p">
        The complete set is defined once in{" "}
        <code className="docs-code">domain/policy.go</code>. A rule outside this
        set is not a valid policy.
      </p>
      <div className="docs-table-wrap">
        <table className="docs-table">
          <thead>
            <tr>
              <th>Rule type</th>
              <th className="docs-table-main">Applies to</th>
            </tr>
          </thead>
          <tbody>
            {ruleTypes.map((rule) => (
              <tr key={rule.type}>
                <td>{rule.type}</td>
                <td className="docs-table-main">{rule.applies}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <h2 className="docs-h2">Enforcement modes</h2>
      <div className="docs-table-wrap">
        <table className="docs-table">
          <thead>
            <tr>
              <th>Mode</th>
              <th className="docs-table-main">Behavior</th>
            </tr>
          </thead>
          <tbody>
            {modes.map((m) => (
              <tr key={m.mode}>
                <td>{m.mode}</td>
                <td className="docs-table-main">{m.behavior}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <h2 className="docs-h2">Rate limits</h2>
      <p className="docs-p">
        <code className="docs-code">RateLimitConfig</code> carries a{" "}
        <code className="docs-code">Key</code>, a <code className="docs-code">Limit</code>,{" "}
        a <code className="docs-code">Window</code>, a <code className="docs-code">Scope</code>,{" "}
        and a <code className="docs-code">BurstAllow</code>. Enforcement uses a
        token bucket with a sliding window, plus a timestamp-set variant for
        stricter burst handling.
      </p>
      <p className="docs-p">A limit is evaluated per scope key, so scopes do not share budget:</p>
      <div className="docs-table-wrap">
        <table className="docs-table">
          <thead>
            <tr>
              <th>Scope</th>
              <th className="docs-table-main">Bucket</th>
            </tr>
          </thead>
          <tbody>
            {scopes.map((s) => (
              <tr key={s.scope}>
                <td>{s.scope}</td>
                <td className="docs-table-main">{s.meaning}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="docs-p">
        A check returns a full decision rather than a bare allow or deny, which
        is what lets a client back off correctly:{" "}
        <code className="docs-code">Allowed</code>,{" "}
        <code className="docs-code">Limit</code>,{" "}
        <code className="docs-code">Remaining</code>,{" "}
        <code className="docs-code">ResetAt</code>,{" "}
        <code className="docs-code">RetryAfter</code>,{" "}
        <code className="docs-code">Scope</code>, and{" "}
        <code className="docs-code">Key</code>.
      </p>

      <h2 className="docs-h2">Budgets</h2>
      <p className="docs-p">
        <code className="docs-code">BudgetConfig</code> adds an independent{" "}
        <code className="docs-code">AlertAt</code> threshold alongside{" "}
        <code className="docs-code">Limit</code>, so warning does not have to
        share a value with denial. <code className="docs-code">BudgetResult</code>{" "}
        reports <code className="docs-code">Current</code>,{" "}
        <code className="docs-code">Projected</code>, and the thresholds, so a
        caller can be warned before the limit is reached.
      </p>

      <h2 className="docs-h2">Model authorization</h2>
      <p className="docs-p">
        Authorization is deny-overrides and is evaluated in a fixed order in{" "}
        <code className="docs-code">policy_authorizer.go</code>:
      </p>
      <ul className="docs-ul">
        <li className="docs-li">
          An empty model name is allowed through; there is nothing to match.
        </li>
        <li className="docs-li">
          A <code className="docs-code">blocklist</code> rule that is enabled and
          lists the model denies the request.
        </li>
        <li className="docs-li">
          An <code className="docs-code">allowlist</code> rule that lists the
          model allows the request.
        </li>
        <li className="docs-li">
          If any enabled allowlist exists and none lists the model, the request
          is denied. An allowlist is therefore exclusive by construction.
        </li>
        <li className="docs-li">
          With no allowlist present at all, the model is permitted.
        </li>
      </ul>
      <p className="docs-p">
        Request inspection failures do not block traffic on their own: a
        transient failure must not take an organization offline, and the
        upstream rejects unknown models regardless.
      </p>

      <h2 className="docs-h2">Policy snapshots</h2>
      <p className="docs-p">
        The gateway never queries Postgres on the hot path. It reads a{" "}
        <code className="docs-code">PolicySnapshot</code> per organization,
        compiled ahead of time and immutable for the life of a request. Each
        snapshot carries an{" "}
        <code className="docs-code">OrganizationID</code>, a monotonic{" "}
        <code className="docs-code">Version</code>, the compiled{" "}
        <code className="docs-code">Rules</code>, the resolved{" "}
        <code className="docs-code">RateLimits</code> and{" "}
        <code className="docs-code">Budgets</code> maps, and{" "}
        <code className="docs-code">EnforcedAt</code>.
      </p>

      <h2 className="docs-h2">Transport defaults</h2>
      <p className="docs-p">
        Set by <code className="docs-code">DefaultGatewayConfig()</code> in{" "}
        <code className="docs-code">interfaces/gatewayv2/types.go</code>.
      </p>
      <div className="docs-table-wrap">
        <table className="docs-table">
          <tbody>
            {defaults.map(([label, value]) => (
              <tr key={label}>
                <td>{label}</td>
                <td className="docs-table-main">{value}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="docs-p">
        Compression is off deliberately: the proxy is byte-exact and never
        rewrites a provider payload on the wire.
      </p>

      <div className="docs-note">
        <strong>Not yet implemented.</strong> Usage events, hourly/daily/monthly
        rollups, retention windows, and privacy modes are not wired up. The
        usage tables and their idempotency constraint exist in the schema, but
        neither data plane writes to them and the async worker runs only health
        probes. See{" "}
        <Link href="/docs/architecture" className="docs-a">
          Architecture
        </Link>{" "}
        for what each process does today.
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
