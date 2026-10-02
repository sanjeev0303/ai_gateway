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
  decision: "Blocked" | "Allowed";
  policy: string;
  reason: string;
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
  headline: "Govern every AI request your company makes.",
  subhead:
    "Sentinel sits between your applications and your model providers. One gateway, every model, and a complete record of who called what, under which policy, at what cost.",
  primary: { label: "Start free", href: "/app/overview" },
  secondary: { label: "See pricing", href: "/pricing" },
  microcopy: "Point one environment variable at the gateway. No card required.",
  stats: [
    { label: "Providers behind one endpoint", value: "8" },
    { label: "Stages recorded per request", value: "12" },
    { label: "Policy evaluated before dispatch", value: "Always" },
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
  headline: "Three hops, one decision point.",
  body: "Your application keeps its existing SDK. Sentinel terminates the call, evaluates policy against the request, forwards to the provider, and writes the record — in that order.",
  inbound: [
    { label: "Your application", detail: "Existing OpenAI-compatible SDK" },
    { label: "Sentinel gateway", detail: "Auth, policy, attribution" },
  ] satisfies readonly FlowStep[],
  outbound: { label: "Model providers", detail: "8 endpoints, one contract" },
  loop: [
    { title: "Authenticate", detail: "Resolves the calling service and project from the key." },
    { title: "Evaluate policy", detail: "Guardrails run before dispatch, not after the bill." },
    { title: "Attribute cost", detail: "Tokens and price are pinned to the request that caused them." },
    { title: "Record the trail", detail: "Append-only entry with the decision and its reason." },
  ] satisfies readonly FlowLoop[],
} as const;

export const PILLARS = {
  eyebrow: "Capabilities",
  headline: "Four surfaces, one request record.",
  body: "Observability, policy, spend, and evidence are not separate products with separate data. They are four views of the same recorded request, so a number in one always matches the others.",
  items: [
    {
      icon: Activity,
      index: "01",
      title: "Observe",
      body: "Every request, streamed and recorded, filterable by project, team, model, and decision.",
      points: ["Live request stream", "Per-conversation traces", "Latency and error breakdown"],
    },
    {
      icon: ShieldCheck,
      index: "02",
      title: "Govern",
      body: "Guardrails evaluated before dispatch, with the matched rule attached to the decision.",
      points: ["Model access control", "Content and PII rules", "Per-team allowlists"],
    },
    {
      icon: Wallet,
      index: "03",
      title: "Attribute",
      body: "Tokens, price, and request counts pinned to the project that caused them.",
      points: ["Per-project cost", "Budget alerts", "Blended cost per MTok"],
    },
    {
      icon: FileClock,
      index: "04",
      title: "Prove",
      body: "An append-only trail that answers what ran, who ran it, and what it was allowed to do.",
      points: ["Immutable audit log", "Exportable records", "Service account history"],
    },
  ] satisfies readonly Pillar[],
} as const;

export const REQUEST_INSPECTOR = {
  eyebrow: "Policy engine",
  headline: "Every decision keeps its reason.",
  body: "A blocked request is only useful if you can see which rule fired and why. Each record carries the policy, the match, the tokens that would have been spent, and the caller that tried.",
  note: "Sample records from the demo environment.",
  samples: [
    {
      id: "req_8f21",
      provider: "Anthropic",
      model: "claude-opus-4",
      project: "support-copilot",
      team: "Customer Experience",
      decision: "Blocked",
      policy: "PII egress",
      reason: "Message contained a 16-digit card number matching the payment pattern.",
      tokens: 3184,
      cost: 0.0955,
      time: "14:19:52",
    },
    {
      id: "req_8f20",
      provider: "OpenAI",
      model: "gpt-4o",
      project: "pricing-service",
      team: "Commerce",
      decision: "Allowed",
      policy: "Cost ceiling",
      reason: "Within the per-request ceiling for the pricing-service project.",
      tokens: 1120,
      cost: 0.0336,
      time: "14:19:41",
    },
    {
      id: "req_8f1e",
      provider: "Google",
      model: "gemini-1.5-pro",
      project: "doc-ingest",
      team: "Platform",
      decision: "Blocked",
      policy: "Model allowlist",
      reason: "Project is scoped to the approved model set and this model is not in it.",
      tokens: 20431,
      cost: 0.2043,
      time: "14:19:12",
    },
    {
      id: "req_8f1b",
      provider: "Mistral",
      model: "mistral-large",
      project: "summarisation",
      team: "Revenue Operations",
      decision: "Allowed",
      policy: "PII egress",
      reason: "No sensitive pattern matched after redaction.",
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
      headline: "Ship against one contract, not eight.",
      body: "Point an existing SDK at the gateway base URL and every provider is reachable without new client code. Model changes become a config edit instead of a release.",
      outcomes: ["One base URL to configure", "Provider failover without code changes", "Errors traced to the request that caused them"],
      surface: [
        { label: "Integration", value: "Base URL swap" },
        { label: "Client changes", value: "None required" },
      ],
    },
    {
      id: "finance",
      label: "Finance",
      headline: "Push the number back to a team.",
      body: "Spend is attributed to the project and team that issued the request, so a chargeback is a report query rather than an allocation argument.",
      outcomes: ["Per-project and per-team cost", "Budget alerts before the overspend", "Blended cost per million tokens"],
      surface: [
        { label: "Attribution", value: "Project and team" },
        { label: "Granularity", value: "Per request" },
      ],
    },
    {
      id: "security",
      label: "Security",
      headline: "Answer with the record, not the log search.",
      body: "Guardrails run before dispatch and every decision is retained with its rule and reason, so control evidence is a query rather than a reconstruction.",
      outcomes: ["Pre-dispatch policy evaluation", "Append-only audit trail", "Exportable request records"],
      surface: [
        { label: "Enforcement", value: "Pre-dispatch" },
        { label: "Evidence", value: "Retained per request" },
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
  headline: "Controls that hold without a service call.",
  body: "Policy is evaluated at the gateway, ahead of the provider, and stored with the request. The controls below are the ones a security review asks about first.",
  capabilities: [
    {
      icon: Layers,
      title: "Layered guardrails",
      body: "Model access, content rules, and PII detection evaluated together, each attributable to the rule that matched.",
    },
    {
      icon: BadgeCheck,
      title: "Model allowlists",
      body: "Approved models are granted per project, so an experimental model cannot be reached from production traffic.",
    },
    {
      icon: Wallet,
      title: "Budgets and ceilings",
      body: "Per-project budgets and per-request ceilings, enforced before dispatch rather than reconciled afterwards.",
    },
    {
      icon: ScrollText,
      title: "Append-only audit log",
      body: "Policy changes, credential rotations, and exports recorded as events that are not edited in place.",
    },
  ] satisfies readonly TrustItem[],
  evidence: [
    { label: "Policy decision", value: "Recorded before dispatch" },
    { label: "Matched rule", value: "Attached to the request" },
    { label: "Actor", value: "Service account or user" },
    { label: "Export", value: "Request-level records" },
  ] satisfies readonly { label: string; value: string }[],
} as const;

export const CODE_TABS = {
  eyebrow: "Integration",
  headline: "One base URL, existing SDK.",
  body: "Sentinel speaks the OpenAI-compatible request contract, so the integration is a base URL and an API key. The snippets below are illustrative.",
  note: "Illustrative snippets. Your gateway host and credentials are issued per environment.",
  samples: [
    {
      id: "curl",
      label: "cURL",
      language: "bash",
      code: [
        "curl https://gateway.sentinel.example/v1/chat/completions \\",
        '  -H "Authorization: Bearer $SENTINEL_API_KEY" \\',
        '  -H "Content-Type: application/json" \\',
        "  -d '{",
        '    "model": "gpt-4o-mini",',
        '    "messages": [{ "role": "user", "content": "Summarise this ticket" }]',
        "  }'",
      ].join("\n"),
    },
    {
      id: "typescript",
      label: "TypeScript",
      language: "typescript",
      code: [
        'import OpenAI from "openai";',
        "",
        "const client = new OpenAI({",
        "  apiKey: process.env.SENTINEL_API_KEY,",
        '  baseURL: "https://gateway.sentinel.example/v1",',
        "});",
        "",
        "const completion = await client.chat.completions.create({",
        '  model: "gpt-4o-mini",',
        '  messages: [{ role: "user", content: "Summarise this ticket" }],',
        "});",
      ].join("\n"),
    },
    {
      id: "python",
      label: "Python",
      language: "python",
      code: [
        "from openai import OpenAI",
        "import os",
        "",
        "client = OpenAI(",
        '    api_key=os.environ["SENTINEL_API_KEY"],',
        '    base_url="https://gateway.sentinel.example/v1",',
        ")",
        "",
        "completion = client.chat.completions.create(",
        '    model="gpt-4o-mini",',
        '    messages=[{"role": "user", "content": "Summarise this ticket"}],',
        ")",
      ].join("\n"),
    },
  ] satisfies readonly CodeSample[],
} as const;

export const ARCHITECTURE = {
  eyebrow: "Data model",
  headline: "What one request leaves behind.",
  body: "A single record carries the caller, the decision, and the cost. Every surface in the product reads this row rather than maintaining its own tally.",
  steps: [
    { label: "Your service", detail: "Calls the gateway with an existing SDK" },
    { label: "Authenticate", detail: "Resolves service account, project, and team" },
    { label: "Evaluate", detail: "Guardrails run against the request payload" },
    { label: "Dispatch", detail: "Forwarded to the selected provider" },
    { label: "Measure", detail: "Tokens, latency, and price recorded" },
    { label: "Retain", detail: "Append-only entry with the decision" },
  ] satisfies readonly FlowStep[],
  fields: [
    { field: "caller", detail: "Service account and the project it belongs to" },
    { field: "model", detail: "Provider, model identifier, and requested parameters" },
    { field: "decision", detail: "Allowed or blocked, with the rule that determined it" },
    { field: "usage", detail: "Prompt and completion tokens, latency, and computed cost" },
    { field: "policy_version", detail: "The policy revision in force at evaluation time" },
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
      sentinel: "Yes",
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
      capability: "Policy before dispatch",
      sentinel: "Evaluated at the gateway",
      inHouse: "Depends on your proxy",
      providerNative: "Per provider, uneven",
    },
    {
      capability: "Audit trail across providers",
      sentinel: "One append-only log",
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
      title: "Every decision is retained",
      body: "Allowed and blocked requests are both recorded with the policy revision in force, so enforcement is reviewable after the fact.",
    },
    {
      icon: FileClock,
      title: "Audit events are append-only",
      body: "Policy edits, credential rotations, and record exports are logged as new events rather than overwriting what came before.",
    },
    {
      icon: Coins,
      title: "Access is scoped per project",
      body: "Grants and budgets attach to a project, so revoking access is a change to one grant rather than a hunt through codebases.",
    },
  ] satisfies readonly TrustItem[],
} as const;

export const PRICING_TEASER = {
  eyebrow: "Pricing",
  headline: "Start on the free tier, move when the traffic does.",
  body: "The free tier covers small teams and does not expire. Paid plans add retention, budget controls, and audit exports.",
  points: [
    "Free for up to five people",
    "No card required to start",
    "Every provider available on all tiers",
  ],
  primary: { label: "See pricing", href: "/pricing" },
  secondary: { label: "Open the dashboard", href: "/app/overview" },
} as const;

export const FINAL_CTA = {
  headline: "Put one gateway in front of your AI traffic.",
  body: "Change a base URL and the first attributed request is on the record. Everything after that is a policy decision you control.",
  primary: { label: "Start free", href: "/app/overview" },
  secondary: { label: "Read about the product", href: "/product" },
} as const;
