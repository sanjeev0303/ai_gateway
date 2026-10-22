import type { LucideIcon } from "lucide-react";
import {
  Activity,
  ArrowLeftRight,
  BadgeDollarSign,
  Ban,
  Bell,
  Blocks,
  Building2,
  CircleDollarSign,
  Code2,
  Cpu,
  FileClock,
  Gauge,
  KeyRound,
  Layers,
  Radar,
  ScrollText,
  Server,
  Shield,
  ShieldCheck,
  Users,
  Wallet,
} from "lucide-react";

export const APP_NAME = "Sentinel";
export const APP_TAGLINE = "Govern every AI request your company makes.";
export const APP_DESCRIPTION =
  "Sentinel is the control plane for enterprise AI: one gateway over your model providers, with per-project cost attribution and a recorded history of every request.";

/** Marketing navigation, rendered by the (marketing) header. */
export const MARKETING_NAV = [
  { label: "Product", href: "/#product" },
  { label: "Solutions", href: "/#solutions" },
  { label: "Pricing", href: "/#pricing" },
  { label: "Resources", href: "/#integration" },
  { label: "Company", href: "/#trust" },
] as const;

export interface NavChild {
  label: string;
  href: string;
  description: string;
}

export interface NavSection {
  title: string;
  icon: LucideIcon;
  href: string;
  children: NavChild[];
}

/** Dashboard sidebar, mirroring the (dashboard)/app route tree. */
export const DASHBOARD_NAV: NavSection[] = [
  {
    title: "Overview",
    icon: Gauge,
    href: "/app/overview",
    children: [],
  },
  {
    title: "Observe",
    icon: Radar,
    href: "/app/observe/live-activity",
    children: [
      {
        label: "Live activity",
        href: "/app/observe/live-activity",
        description: "Streaming request feed",
      },
      {
        label: "Usage",
        href: "/app/observe/usage",
        description: "Tokens and call volume",
      },
      {
        label: "Costs",
        href: "/app/observe/costs",
        description: "Spend and forecast",
      },
      {
        label: "Models",
        href: "/app/observe/models",
        description: "Catalog and health",
      },
      {
        label: "Providers",
        href: "/app/observe/providers",
        description: "Upstream status",
      },
      {
        label: "Users",
        href: "/app/observe/users",
        description: "People and activity",
      },
      {
        label: "Teams",
        href: "/app/observe/teams",
        description: "Groups and spend",
      },
      {
        label: "Projects",
        href: "/app/observe/projects",
        description: "Workloads",
      },
      {
        label: "Conversations",
        href: "/app/observe/conversations",
        description: "Traces and turns",
      },
    ],
  },
  {
    title: "Govern",
    icon: ShieldCheck,
    href: "/app/govern/policies",
    children: [
      {
        label: "Policies",
        href: "/app/govern/policies",
        description: "Enforcement rules",
      },
      {
        label: "Model access",
        href: "/app/govern/model-access",
        description: "Grants and denials",
      },
      {
        label: "Budgets",
        href: "/app/govern/budgets",
        description: "Limits and actions",
      },
      {
        label: "Guardrails",
        href: "/app/govern/guardrails",
        description: "Input and output filters",
      },
      {
        label: "Credentials",
        href: "/app/govern/credentials",
        description: "Keys and rotation",
      },
    ],
  },
  {
    title: "Security",
    icon: Shield,
    href: "/app/security/audit-logs",
    children: [
      {
        label: "Audit logs",
        href: "/app/security/audit-logs",
        description: "Immutable history",
      },
      {
        label: "Alerts",
        href: "/app/security/alerts",
        description: "Open incidents",
      },
      {
        label: "Events",
        href: "/app/security/events",
        description: "Platform stream",
      },
    ],
  },
  {
    title: "Platform",
    icon: Server,
    href: "/app/platform/provider-health",
    children: [
      {
        label: "Provider health",
        href: "/app/platform/provider-health",
        description: "Upstream reachability",
      },
      {
        label: "Provider accounts",
        href: "/app/platform/provider-accounts",
        description: "Upstream connections",
      },
      {
        label: "Service accounts",
        href: "/app/platform/service-accounts",
        description: "Machine identities",
      },
      {
        label: "Gateway credentials",
        href: "/app/platform/gateway-credentials",
        description: "Keys and rotation",
      },
      {
        label: "Model registry",
        href: "/app/platform/model-registry",
        description: "Enabled models",
      },
      {
        label: "Pricing management",
        href: "/app/platform/pricing",
        description: "Token rates and price sheets",
      },
    ],
  },
  {
    title: "Organization",
    icon: Building2,
    href: "/app/organization/members",
    children: [
      {
        label: "Members",
        href: "/app/organization/members",
        description: "People and roles",
      },
      {
        label: "Invitations",
        href: "/app/organization/invitations",
        description: "Pending access",
      },
      {
        label: "Teams",
        href: "/app/organization/teams",
        description: "Groups and budgets",
      },
      {
        label: "Projects",
        href: "/app/organization/projects",
        description: "Workloads and owners",
      },
      {
        label: "Settings",
        href: "/app/organization/settings",
        description: "Organization profile",
      },
    ],
  },
  {
    title: "Billing",
    icon: Wallet,
    href: "/app/billing/usage",
    children: [
      {
        label: "Usage",
        href: "/app/billing/usage",
        description: "Meters and limits",
      },
      {
        label: "Subscription",
        href: "/app/billing/subscription",
        description: "Plan and add-ons",
      },
      {
        label: "Invoices",
        href: "/app/billing/invoices",
        description: "Statements and PDFs",
      },
    ],
  },
  {
    title: "Developer",
    icon: Code2,
    href: "/app/developer/quick-start",
    children: [
      {
        label: "Quick start",
        href: "/app/developer/quick-start",
        description: "First request in minutes",
      },
      {
        label: "SDKs",
        href: "/app/developer/sdks",
        description: "Typed client libraries",
      },
      {
        label: "CLI",
        href: "/app/developer/cli",
        description: "Command reference",
      },
      {
        label: "API reference",
        href: "/app/developer/api-docs",
        description: "Endpoints and schemas",
      },
    ],
  },
];

/** Quick-access tiles for the overview page. */
export const OVERVIEW_TILES = [
  {
    label: "Live requests",
    href: "/app/observe/live-activity",
    icon: Activity,
    description: "Watch traffic as it happens",
  },
  {
    label: "Spend this month",
    href: "/app/observe/costs",
    icon: CircleDollarSign,
    description: "Attribution and forecast",
  },
  {
    label: "Policy blocks",
    href: "/app/govern/policies",
    icon: Ban,
    description: "What was stopped and why",
  },
  {
    label: "Open alerts",
    href: "/app/security/alerts",
    icon: Bell,
    description: "Triage active incidents",
  },
  {
    label: "Model catalog",
    href: "/app/observe/models",
    icon: Cpu,
    description: "Every model behind one key",
  },
  {
    label: "Provider status",
    href: "/app/observe/providers",
    icon: ArrowLeftRight,
    description: "Upstream health and latency",
  },
  {
    label: "Audit trail",
    href: "/app/security/audit-logs",
    icon: ScrollText,
    description: "Who changed what, and when",
  },
  {
    label: "Credentials",
    href: "/app/govern/credentials",
    icon: KeyRound,
    description: "Rotation and expiry",
  },
] as const;

/** Marketing feature grid on the home page. */
export const MARKETING_FEATURES = [
  {
    icon: Blocks,
    title: "One gateway, every model",
    description:
      "Route provider traffic through a single endpoint. Request bodies are forwarded as sent, with auth headers rewritten for Anthropic and OpenAI upstreams, so the gateway never imposes a schema of its own.",
  },
  {
    icon: ShieldCheck,
    title: "Governed in the request path",
    description:
      "Scoped credentials and compiled, immutable policy snapshots authorize, rate limit, and budget check every request before it is dispatched upstream.",
  },
  {
    icon: BadgeDollarSign,
    title: "Usage you can price",
    description:
      "Every request records model, tokens, and latency off the hot path. A worker prices requests from the price sheet effective at request time and rolls them into hourly, daily, and monthly buckets.",
  },
  {
    icon: FileClock,
    title: "An audit trail you can export",
    description:
      "Tenant, credential, policy, and configuration changes are recorded as they happen and can be exported on demand.",
  },
  {
    icon: Users,
    title: "Built for real orgs",
    description:
      "Isolation across organizations, teams, and projects, with credentials scoped to gateway, control, or both.",
  },
  {
    icon: Layers,
    title: "Works with your stack",
    description:
      "The gateway speaks the Anthropic request contract, so existing Anthropic client code works by changing the base URL and the key. No wrapper SDK to adopt.",
  },
];

/** No billing or settlement exists yet, so there is no price list to publish. */
export const PRICING_TIERS = [
  {
    name: "Design partners",
    monthlyPrice: null,
    tagline:
      "The platform is pre-release. We are onboarding teams directly rather than selling seats.",
    cta: "Get in touch",
    features: [
      "Self-serve control plane over the REST API",
      "Gateway hot path: scoped credentials, policy snapshots, rate limits, budgets",
      "Usage accounting with idempotent ingest and hourly, daily, and monthly rollups",
      "Audit export for tenant, credential, and policy changes",
      "Anthropic-compatible request surface, auth headers rewritten for Anthropic and OpenAI",
    ],
  },
] as const;

/** Default date range offered by the dashboard range picker. */
export const DATE_RANGES = ["1h", "24h", "7d", "30d", "90d"] as const;
export type DateRange = (typeof DATE_RANGES)[number];

export const MARKETING_PROVIDERS = [
  "OpenAI",
  "Anthropic",
  "Google",
  "Mistral",
  "Meta",
  "AWS Bedrock",
  "Azure AI",
  "Cohere",
] as const;

export const MARKETING_SHOWCASE = [
  {
    icon: Gauge,
    title: "Observe",
    description:
      "Every request is recorded with its model, tokens, and latency. Records are written off the hot path, so accounting never slows dispatch.",
  },
  {
    icon: ShieldCheck,
    title: "Govern",
    description:
      "Compiled, immutable policy snapshots authorize each request in the path, alongside rate limits and budget checks. Enforcement runs on every request; there is no API for authoring the rules yet.",
  },
  {
    icon: CircleDollarSign,
    title: "Price it",
    description:
      "A worker prices each request from the price sheet that was effective when it ran, then rolls it into hourly, daily, and monthly buckets. Nothing settles an invoice yet.",
  },
  {
    icon: ScrollText,
    title: "Prove it",
    description:
      "Export the audit trail on demand for tenant, credential, and policy changes. Retention windows are not implemented yet.",
  },
] as const;

export const MARKETING_SECURITY = [
  {
    icon: Users,
    title: "Tenant isolation",
    description:
      "Organizations, teams, and projects each get scoped access. A credential resolves to exactly one tenant context.",
  },
  {
    icon: KeyRound,
    title: "Scoped credentials",
    description:
      "Keys are limited to gateway, control, or both, and every gateway request carries a request id you can trace.",
  },
  {
    icon: Layers,
    title: "Nothing reshaped on the wire",
    description:
      "Request bodies are forwarded as sent, so the gateway never normalizes your request into a schema it would have to guess at.",
  },
  {
    icon: ScrollText,
    title: "Exportable audit trail",
    description:
      "Credential, policy, and tenant changes are recorded and can be exported on demand.",
  },
] as const;

export const MARKETING_PERSONAS = [
  {
    icon: Code2,
    title: "Platform engineers",
    description:
      "Route provider traffic through one endpoint without rewriting clients. Request bodies are forwarded as sent, so switching models is a request parameter.",
  },
  {
    icon: Shield,
    title: "Security and compliance",
    description:
      "Issue credentials scoped to gateway or control, and export the audit trail when someone asks what changed and who changed it.",
  },
  {
    icon: CircleDollarSign,
    title: "Finance",
    description:
      "Per-request token and latency records, priced from the price sheet that was effective when the request ran, then rolled up by hour, day, and month.",
  },
  {
    icon: Users,
    title: "Engineering leadership",
    description:
      "See which teams and projects call which models, and what each call cost to run.",
  },
] as const;
