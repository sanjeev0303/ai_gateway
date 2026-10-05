import {
  Activity,
  BadgeCheck,
  Coins,
  FileClock,
  Layers,
  Receipt,
  Route,
  Scale,
  ScrollText,
  ShieldCheck,
  Split,
  Wallet,
} from "lucide-react";

export interface HeroStat {
  label: string;
  value: string;
}

export interface ProblemPoint {
  icon: typeof Activity;
  title: string;
  body: string;
}

export interface FlowStep {
  label: string;
  detail: string;
}

export interface FlowLoop {
  title: string;
  detail: string;
}

export interface Pillar {
  icon: typeof Activity;
  index: string;
  title: string;
  body: string;
  points: readonly string[];
}

export interface RequestSample {
  id: string;
  provider: string;
  model: string;
  project: string;
  team: string;
  outcome: "Success" | "Error";
  status: string;
  detail: string;
  tokens: number;
  cost: number;
  time: string;
}

export interface UseCase {
  id: string;
  label: string;
  headline: string;
  body: string;
  outcomes: readonly string[];
  surface: { label: string; value: string }[];
}

export type AttributionMetric = "spend" | "tokens" | "requests";

export interface AttributionRow {
  label: string;
  spend: number;
  tokens: number;
  requests: number;
}

export interface AttributionMetricCopy {
  id: AttributionMetric;
  label: string;
  caption: string;
  unit: string;
  format: (value: number) => string;
}

export interface CodeSample {
  id: string;
  label: string;
  language: string;
  code: string;
}

export interface RecordField {
  field: string;
  detail: string;
}

export interface ComparisonRow {
  capability: string;
  sentinel: string;
  inHouse: string;
  providerNative: string;
}

export interface TrustItem {
  icon: typeof Activity;
  title: string;
  body: string;
}

export const HERO = {
  eyebrow: "AI control plane",
  headline: "See what your AI requests actually cost.",
  subhead:
    "Sentinel sits between your applications and your model providers. One gateway, and a record of every request: which model, how many tokens, what it cost, and which project it belongs to.",
  primary: { label: "Open the dashboard", href: "/app/overview" },
  secondary: { label: "See pricing", href: "/pricing" },
  microcopy: "Point one environment variable at the gateway. No card required.",
  stats: [
    { label: "Provider kinds supported", value: "5" },
    { label: "Spend attributed to", value: "The project" },
    { label: "Request formats accepted", value: "Anthropic messages" },
  ] satisfies readonly HeroStat[],
} as const;

export const PROBLEM = {
  eyebrow: "The problem",
  headline: "Model sprawl outruns the way you buy it.",
  body: "Every team reaches for a different provider with a different key, a different invoice, and a different idea of what is allowed. By the time finance asks what the number is, nobody can answer from the request itself.",
  points: [
    {
      icon: Split,
      title: "Keys scattered across services",
      body: "Provider credentials end up in environment variables, notebooks, and CI secrets. Nobody owns the full list, so revoking one means finding every copy of it.",
    },
    {
      icon: Receipt,
      title: "Cost arrives without attribution",
      body: "A provider invoice tells you what you spent. It does not tell you which team, project, or feature caused it, so the number cannot be pushed back to a team.",
    },
    {
      icon: Scale,
      title: "Policy lives in application code",
      body: "Guardrails get reimplemented per service. Two teams enforcing the same rule write it differently, and a rule change becomes a multi-repo migration.",
    },
    {
      icon: ScrollText,
      title: "No answer for an auditor",
      body: "When someone asks which model touched a record last quarter, the honest answer is a log search across four providers and a hopeful guess.",
    },
  ] satisfies readonly ProblemPoint[],
} as const;

export const GATEWAY_FLOW = {
  eyebrow: "How it works",
  headline: "Authenticate, dispatch, record.",
  body: "Your application sends an Anthropic-style messages request. Sentinel authenticates the credential, authorizes the model against policy, applies rate and budget limits, forwards the request body with the auth headers the target provider expects, and writes the resulting usage to a tenant-scoped record.",
  inbound: [
    { label: "Your application", detail: "Anthropic-style /v1/messages call" },
    { label: "Sentinel gateway", detail: "Credential auth, attribution" },
  ] satisfies readonly FlowStep[],
  outbound: { label: "Model providers", detail: "Five provider kinds" },
  loop: [
    { title: "Authenticate", detail: "Resolves the credential, its scope, and the calling project." },
    { title: "Dispatch", detail: "Forwards the request body upstream, rewriting only the auth and version headers." },
    { title: "Measure", detail: "Tokens, latency, and derived cost recorded off the request path." },
    { title: "Attribute", detail: "Usage rolls up by project, model, and provider." },
  ] satisfies readonly FlowLoop[],
} as const;

export const PILLARS = {
  eyebrow: "Capabilities",
  headline: "Four views of the same recorded request.",
  body: "Usage, cost, credentials, and audit history all read from the same tenant-scoped data, so a number in one always matches the others.",
  items: [
    {
      icon: Activity,
      index: "01",
      title: "Observe",
      body: "Request counts, token totals, error rate, and latency percentiles over a selected window.",
      points: ["Usage summary and timeseries", "Latency percentiles (p50/p95)", "Breakdown by project and model"],
    },
    {
      icon: Wallet,
      index: "02",
      title: "Attribute",
      body: "Tokens and derived cost pinned to the project that issued the request.",
      points: ["Per-project cost and tokens", "Share of total spend", "Blended cost per MTok"],
    },
    {
      icon: ShieldCheck,
      index: "03",
      title: "Control",
      body: "Scoped credentials with provider keys held by the gateway. Every request is authorized against a compiled policy snapshot; guardrail evaluation is not implemented yet.",
      points: ["Per-project credentials", "Provider key custody", "Credential rotation"],
    },
    {
      icon: FileClock,
      index: "04",
      title: "Prove",
      body: "Control-plane changes are recorded as events, so you can see what changed and who changed it.",
      points: ["Audit event history", "Credential and project changes", "Exportable records"],
    },
  ] satisfies readonly Pillar[],
} as const;

export const REQUEST_INSPECTOR = {
  eyebrow: "Request records",
  headline: "What one request leaves behind.",
  body: "Each request is recorded with the model, provider, project, token counts, outcome, and derived cost. There is no policy decision attached, because no policy evaluation happens at the gateway today.",
  note: "Sample records shaped like the fields the gateway records. Costs are illustrative.",
  samples: [
    {
      id: "req_8f21",
      provider: "anthropic",
      model: "claude-sonnet-4",
      project: "support-copilot",
      team: "Customer Experience",
      outcome: "Success",
      status: "200",
      detail: "Forwarded to the provider and returned in 1.8s. Tokens and cost attributed to support-copilot.",
      tokens: 3184,
      cost: 0.0955,
      time: "14:19:52",
    },
    {
      id: "req_8f20",
      provider: "openai",
      model: "gpt-4o",
      project: "pricing-service",
      team: "Commerce",
      outcome: "Success",
      status: "200",
      detail: "Forwarded to the provider and returned in 0.9s. Counted against pricing-service.",
      tokens: 1120,
      cost: 0.0336,
      time: "14:19:41",
    },
    {
      id: "req_8f1e",
      provider: "gemini",
      model: "gemini-2.0-flash",
      project: "doc-ingest",
      team: "Platform",
      outcome: "Error",
      status: "429",
      detail: "Upstream rate limit returned to the caller. The failure is recorded against doc-ingest.",
      tokens: 0,
      cost: 0,
      time: "14:19:12",
    },
    {
      id: "req_8f1b",
      provider: "azure_openai",
      model: "gpt-4o-mini",
      project: "summarisation",
      team: "Revenue Operations",
      outcome: "Success",
      status: "200",
      detail: "Forwarded to the provider and returned in 2.3s. Counted against summarisation.",
      tokens: 5402,
      cost: 0.0918,
      time: "14:18:57",
    },
  ] satisfies readonly RequestSample[],
} as const;

export const USE_CASES = {
  eyebrow: "Who it is for",
  headline: "Different mandates, same request log.",
  body: "Each team arrives with a different question. They all end up reading the same record.",
  cases: [
    {
      id: "engineering",
      label: "Engineering",
      headline: "One base URL, one credential.",
      body: "Point your client at the gateway base URL and send Anthropic-style messages requests. Provider selection stays a gateway-side configuration concern rather than a change in your calling code.",
      outcomes: ["One base URL to configure", "Per-project credentials", "Usage recorded per request"],
      surface: [
        { label: "Integration", value: "Base URL swap" },
        { label: "Request format", value: "Anthropic messages" },
      ],
    },
    {
      id: "finance",
      label: "Finance",
      headline: "Push the number back to a project.",
      body: "Spend is attributed to the project that issued the request, so an allocation question becomes a report query rather than an argument. Billing and chargeback are not built yet — these are reported costs, not invoices.",
      outcomes: ["Per-project cost and tokens", "Share of total spend", "Cost over any window"],
      surface: [
        { label: "Attribution", value: "Per project" },
        { label: "Granularity", value: "Per request" },
      ],
    },
    {
      id: "security",
      label: "Security",
      headline: "Know which keys touched what.",
      body: "Provider keys stay with the gateway, your services hold project-scoped Sentinel credentials, and control-plane changes are recorded as audit events. Every request is authorized against the organization's compiled policy before it reaches a provider.",
      outcomes: ["Provider keys held by the gateway", "Model allow and block rules enforced per request", "Audit events for config changes"],
      surface: [
        { label: "Key custody", value: "Gateway-side" },
        { label: "Request policy", value: "Model allow/block, per request" },
      ],
    },
  ] satisfies readonly UseCase[],
} as const;

export const ATTRIBUTION = {
  eyebrow: "Cost attribution",
  headline: "The same traffic, read three ways.",
  body: "Switch the metric and the same recorded requests re-aggregate. Spend, tokens, and request counts all derive from the request record, so they can never drift apart.",
  note: "Sample attribution across teams, demo environment.",
  metrics: [
    {
      id: "spend",
      label: "Spend",
      caption: "Cost pinned to the team that issued the request.",
      unit: "USD",
      format: (value: number) => `$${value.toLocaleString("en-US", { maximumFractionDigits: 0 })}`,
    },
    {
      id: "tokens",
      label: "Tokens",
      caption: "Total tokens processed, in millions.",
      unit: "MTok",
      format: (value: number) => `${value.toLocaleString("en-US")}M`,
    },
    {
      id: "requests",
      label: "Requests",
      caption: "Count of recorded requests in the window.",
      unit: "requests",
      format: (value: number) => value.toLocaleString("en-US"),
    },
  ] satisfies readonly AttributionMetricCopy[],
  rows: [
    { label: "Customer Experience", spend: 4820, tokens: 61, requests: 184200 },
    { label: "Platform", spend: 3940, tokens: 48, requests: 142600 },
    { label: "Commerce", spend: 2610, tokens: 33, requests: 96800 },
    { label: "Revenue Operations", spend: 1740, tokens: 22, requests: 74300 },
    { label: "Data Science", spend: 1090, tokens: 14, requests: 38900 },
  ] satisfies readonly AttributionRow[],
} as const;

export const GOVERNANCE = {
  eyebrow: "Governance",
  headline: "The controls that exist, and the one that does not.",
  body: "Credential scoping, key custody, cost attribution, and an audit log are built today. Request-level policy evaluation is not — we would rather name the gap than imply it is closed.",
  capabilities: [
    {
      icon: BadgeCheck,
      title: "Scoped credentials",
      body: "Services authenticate with a Sentinel credential scoped to a single project, so a leaked key is traceable to one project rather than the whole platform.",
    },
    {
      icon: Layers,
      title: "Provider key custody",
      body: "Upstream provider keys are held by the gateway and never returned to the calling application. Services only ever hold a Sentinel key.",
    },
    {
      icon: Wallet,
      title: "Spend attribution",
      body: "Every request carries the project that issued it, so cost is attributed at write time instead of reconciled from an invoice afterwards.",
    },
    {
      icon: ScrollText,
      title: "Control-plane audit log",
      body: "Credential, project, and configuration changes are recorded as events you can list and export.",
    },
  ] satisfies readonly TrustItem[],
  evidence: [
    { label: "Audit events", value: "Control-plane changes" },
    { label: "Actor", value: "Recorded per event" },
    { label: "Request record", value: "Model, tokens, cost, latency" },
    { label: "Export", value: "Audit records" },
    { label: "Policy decision", value: "Evaluated per request" },
  ] satisfies readonly { label: string; value: string }[],
} as const;

export const CODE_TABS = {
  eyebrow: "Integration",
  headline: "One base URL, one header.",
  body: "The gateway forwards request bodies as sent, so there is no wrapper SDK to adopt. Send the request with any HTTP client.",
  note: "Replace $GATEWAY_URL with your deployment's gateway host. Credentials are issued per environment.",
  samples: [
    {
      id: "curl",
      label: "cURL",
      language: "bash",
      code: [
        "curl $GATEWAY_URL/v1/messages \\",
        '  -H "x-api-key: $AIGW_API_KEY" \\',
        '  -H "content-type: application/json" \\',
        "  -d '{",
        '    "model": "claude-sonnet-4-5",',
        '    "max_tokens": 1024,',
        '    "messages": [{ "role": "user", "content": "Summarise this ticket" }]',
        "  }'",
      ].join("\n"),
    },
    {
      id: "typescript",
      label: "TypeScript",
      language: "typescript",
      code: [
        "const response = await fetch(`${process.env.GATEWAY_URL}/v1/messages`, {",
        '  method: "POST",',
        "  headers: {",
        '    "x-api-key": process.env.AIGW_API_KEY!,',
        '    "content-type": "application/json",',
        "  },",
        "  body: JSON.stringify({",
        '    model: "claude-sonnet-4-5",',
        "    max_tokens: 1024,",
        '    messages: [{ role: "user", content: "Summarise this ticket" }],',
        "  }),",
        "});",
        "",
        "const message = await response.json();",
      ].join("\n"),
    },
    {
      id: "python",
      label: "Python",
      language: "python",
      code: [
        "import os",
        "import httpx",
        "",
        'url = os.environ["GATEWAY_URL"] + "/v1/messages"',
        "",
        "response = httpx.post(",
        "    url,",
        '    headers={"x-api-key": os.environ["AIGW_API_KEY"]},',
        "    json={",
        '        "model": "claude-sonnet-4-5",',
        '        "max_tokens": 1024,',
        '        "messages": [{"role": "user", "content": "Summarise this ticket"}],',
        "    },",
        ")",
        "",
        "print(response.json())",
      ].join("\n"),
    },
  ] satisfies readonly CodeSample[],
} as const;

export const ARCHITECTURE = {
  eyebrow: "Data model",
  headline: "What one request leaves behind.",
  body: "A request record carries the caller and the usage. Reporting reads these records instead of maintaining its own tally.",
  steps: [
    { label: "Your service", detail: "Calls the gateway over HTTP with a scoped credential" },
    { label: "Authenticate", detail: "Resolves the credential, its scope, tenant, and project" },
    { label: "Dispatch", detail: "Forwarded with the auth headers the target provider expects" },
    { label: "Measure", detail: "Tokens, latency, and derived cost recorded off the request path" },
    { label: "Attribute", detail: "Usage rolls up by project, model, and provider" },
  ] satisfies readonly FlowStep[],
  fields: [
    { field: "caller", detail: "The credential and the project it belongs to" },
    { field: "model", detail: "Provider kind, model identifier, and requested parameters" },
    { field: "outcome", detail: "Success or failure, with the recorded status" },
    { field: "usage", detail: "Prompt and completion tokens, latency, and derived cost" },
    { field: "tenant", detail: "Tenant and project the usage rolls up into" },
  ] satisfies readonly RecordField[],
} as const;

export const COMPARISON = {
  eyebrow: "Alternatives",
  headline: "Three ways to answer the same question.",
  body: "Most teams arrive from one of these directions. Each works until the questions get harder.",
  columns: ["Sentinel", "Built in-house", "Provider-native tooling"],
  rows: [
    {
      capability: "Providers behind one endpoint",
      sentinel: "Five provider kinds",
      inHouse: "Yes, and maintained by you",
      providerNative: "One provider at a time",
    },
    {
      capability: "Cost attributed to a project",
      sentinel: "Per request, at write time",
      inHouse: "Possible, once the pipeline exists",
      providerNative: "Invoice level only",
    },
    {
      capability: "Request-level policy",
      sentinel: "Model allow/block, per request",
      inHouse: "Depends on your proxy",
      providerNative: "Per provider, uneven",
    },
    {
      capability: "Usage record across providers",
      sentinel: "One tenant-scoped record",
      inHouse: "Whatever you retain",
      providerNative: "Siloed per provider",
    },
    {
      capability: "Time to first attributed request",
      sentinel: "One environment variable",
      inHouse: "Build, then maintain",
      providerNative: "Immediate, partial coverage",
    },
  ] satisfies readonly ComparisonRow[],
} as const;

export const TRUST = {
  eyebrow: "Trust",
  headline: "What we can show you today.",
  body: "These are mechanisms already present in the product, not aspirations. Where something is not built yet, we would rather say so than imply otherwise.",
  items: [
    {
      icon: Route,
      title: "Credentials never reach the app",
      body: "Provider keys live with the gateway. Your services hold a Sentinel key scoped to one project, so a leak does not expose a provider account.",
    },
    {
      icon: ScrollText,
      title: "Requests leave a record",
      body: "Each request is recorded with its model, provider, token counts, outcome, and latency, written off the request path so accounting never slows dispatch.",
    },
    {
      icon: FileClock,
      title: "Configuration changes are logged",
      body: "Credential, project, and configuration changes are recorded as audit events you can list and export.",
    },
    {
      icon: Coins,
      title: "Credentials are scoped per project",
      body: "A Sentinel credential belongs to one project, so revoking access is a single credential change. Rate limits are enforced per credential and spend caps per organization, on every request.",
    },
  ] satisfies readonly TrustItem[],
} as const;

export const PRICING_TEASER = {
  eyebrow: "Pricing",
  headline: "Not priced yet.",
  body: "There is no billing or seat management in the product yet, so any price list would be fiction. What the platform does today is on the pricing page.",
  points: [
    "No card required, because there is no checkout",
    "No seat limits, because nothing is metered per seat",
    "No provider held back, because nothing is gated behind a plan",
  ],
  primary: { label: "See what exists", href: "/pricing" },
  secondary: { label: "Open the dashboard", href: "/app/overview" },
} as const;

export const FINAL_CTA = {
  headline: "Put one gateway in front of your AI traffic.",
  body: "Change a base URL and your first request is on the record, with its model, provider, token counts, outcome, and latency. Everything after it is scoped to one tenant.",
  primary: { label: "Open the dashboard", href: "/app/overview" },
  secondary: { label: "Read about the product", href: "/product" },
} as const;
