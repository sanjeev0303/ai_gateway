export type Control = {
  readonly name: string;
  readonly summary: string;
  readonly values: readonly string[];
  readonly source: string;
};

export const controls: readonly Control[] = [
  {
    name: "Policy rule types",
    summary: "The complete rule vocabulary a policy can contain.",
    values: [
      "rate_limit",
      "token_budget",
      "budget",
      "blocklist",
      "allowlist",
      "content_filter",
    ],
    source: "domain.RuleType*",
  },
  {
    name: "Enforcement modes",
    summary: "How aggressively a compiled policy gates live traffic.",
    values: ["permissive", "active", "disabled"],
    source: "domain.Enforcement*",
  },
  {
    name: "Rate limit scopes",
    summary:
      "A limit is evaluated per scope key, so one credential cannot spend another scope's budget.",
    values: ["organization", "project", "credential", "user"],
    source: "gatewayv2.RateLimitConfig.Scope",
  },
  {
    name: "Rate limit algorithm",
    summary:
      "Token bucket with sliding window, plus a timestamp-set variant for stricter burst handling.",
    values: ["token bucket", "sliding window"],
    source: "gatewayv2.RateLimiter",
  },
  {
    name: "Rate limit result",
    summary: "Every check returns a full decision, not a bare allow or deny.",
    values: ["limit", "remaining", "reset_at", "retry_after"],
    source: "gatewayv2.RateLimitResult",
  },
  {
    name: "Default rate limit",
    summary:
      "Applied when a snapshot carries no explicit rate_limit rule for a scope.",
    values: ["1000 requests", "per 1 minute"],
    source: "DefaultGatewayConfig",
  },
  {
    name: "Budget controls",
    summary:
      "Spend is checked against a limit and window, with an independent alert threshold.",
    values: ["limit", "window", "scope", "alert_at"],
    source: "gatewayv2.BudgetConfig",
  },
  {
    name: "Default budget",
    summary: "Applied when a snapshot carries no explicit budget rule for a scope.",
    values: ["1000.00", "per 1 hour"],
    source: "DefaultGatewayConfig",
  },
  {
    name: "Model authorization",
    summary:
      "Deny-overrides. A blocklist match denies, an allowlist that omits the model denies, no matching rule permits.",
    values: ["deny-overrides", "provider-qualified ids"],
    source: "gatewayv2.PolicySnapshot.Authorize",
  },
  {
    name: "Policy snapshots",
    summary:
      "Compiled per organization, versioned, and immutable for the life of a request.",
    values: ["versioned", "immutable", "cached in hot path"],
    source: "gatewayv2.PolicySnapshot",
  },
  {
    name: "Usage ledger",
    summary:
      "Schema and repository ship today, with a database-level unique key on (organization, request_id, event_type) that makes ingest idempotent. The gateway does not write events on the request path yet.",
    values: ["append-only table", "idempotent by constraint", "not on hot path yet"],
    source: "migrations/00007_usage.sql",
  },
  {
    name: "Usage rollups",
    summary:
      "Hourly, daily, and monthly aggregate tables exist. The async worker currently serves only health probes, so nothing populates them yet.",
    values: ["hourly", "daily", "monthly", "not computed yet"],
    source: "migrations/00007_usage.sql",
  },
  {
    name: "Request timeouts",
    summary: "Client, upstream, and idle deadlines for proxied traffic.",
    values: ["30s request", "5m upstream", "90s idle"],
    source: "DefaultGatewayConfig",
  },
  {
    name: "Concurrency",
    summary: "Bounded so a burst queues instead of exhausting the process.",
    values: ["10000 queued", "200 per host", "HTTP/2 on"],
    source: "DefaultGatewayConfig",
  },
  {
    name: "Payload fidelity",
    summary:
      "Compression is off so the proxy stays byte-exact. Provider payloads are never rewritten.",
    values: ["byte-exact", "compression off"],
    source: "DefaultGatewayConfig",
  },
];

export const deployment = {
  name: "Self-hosted",
  price: "Your infrastructure",
  summary:
    "The whole stack runs on hardware you control: data plane, control plane, async worker, and dashboard. The gateway makes no telemetry calls and has no hosted dependency, and traffic leaves your network only for the provider endpoints you configure.",
  points: [
    "gateway, gateway-v2, control-api, worker, and web in one stack",
    "Postgres, Redis, and NATS wired together by the compose file",
    "Compiled policy snapshots evaluated on your own CPUs",
    "Usage and audit rows stay in your database",
    "Docker Compose, Kubernetes, and Terraform manifests in the repository",
    "Every row in the table above is gateway behavior, not a plan feature",
  ],
  cta: "Read the deployment guide",
  href: "/docs/deployment",
} as const;

export const hosted = {
  name: "Hosted",
  price: "Not yet available",
  summary:
    "Hosted plans are designed but not built. The plan, subscription, and billing event tables exist in the schema and carry no seeded plans, prices, or entitlements, so there is nothing to quote yet. Self-hosting is the supported path today.",
  cta: "Deploy it yourself",
  href: "/docs/deployment",
} as const;
