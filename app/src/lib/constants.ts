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
  Lock,
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
  "Sentinel is the control plane for enterprise AI: one gateway over every model provider, with policy enforcement, cost attribution, and an immutable audit trail.";

/** Marketing navigation, rendered by the (marketing) header. */
export const MARKETING_NAV = [
  { label: "Product", href: "/product" },
  { label: "Solutions", href: "/solutions" },
  { label: "Pricing", href: "/pricing" },
  { label: "Resources", href: "/resources" },
  { label: "Company", href: "/company" },
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
        description: "Uptime and latency",
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
      "Route traffic to any provider behind a single endpoint. Swap models without redeploying, and fail over automatically when an upstream degrades.",
  },
  {
    icon: ShieldCheck,
    title: "Policy at the edge",
    description:
      "Enforce PII redaction, injection defence, and residency rules on every request in under 40ms, before tokens are ever billed.",
  },
  {
    icon: BadgeDollarSign,
    title: "Cost you can defend",
    description:
      "Attribute every dollar to a user, team, project, and model. Set budgets that alert, throttle, or hard-stop automatically.",
  },
  {
    icon: FileClock,
    title: "An audit trail that holds up",
    description:
      "Immutable, exportable records of every request, policy decision, and configuration change — retained for seven years.",
  },
  {
    icon: Users,
    title: "Built for real orgs",
    description:
      "SCIM provisioning, SAML SSO, role-based access, and per-team isolation. Your security team signs off once, not every quarter.",
  },
  {
    icon: Layers,
    title: "Works with your stack",
    description:
      "Native SDKs for TypeScript and Python, an OpenAI-compatible API, and drop-in proxy support for LangChain, LlamaIndex, and Vercel AI SDK.",
  },
];

/** Pricing tiers rendered on /pricing. */
export const PRICING_TIERS = [
  {
    name: "Starter",
    monthlyPrice: 0,
    tagline: "For a first team getting a handle on AI spend.",
    cta: "Start free",
    features: [
      "Up to 5 team members",
      "1 gateway project",
      "Community model providers",
      "7-day audit log retention",
      "Email support",
    ],
  },
  {
    name: "Growth",
    monthlyPrice: 499,
    tagline: "For teams shipping AI to production.",
    cta: "Start 14-day trial",
    featured: true,
    features: [
      "Unlimited team members",
      "Unlimited projects and environments",
      "Policy engine and guardrails",
      "Budget enforcement and alerts",
      "90-day audit log retention",
      "SSO and role-based access",
    ],
  },
  {
    name: "Enterprise",
    monthlyPrice: null,
    tagline: "For regulated orgs with procurement requirements.",
    cta: "Talk to sales",
    features: [
      "Everything in Growth",
      "Private networking and data residency",
      "SCIM provisioning",
      "Custom retention up to 7 years",
      "99.99% uptime SLA",
      "Dedicated support engineer",
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
      "Every request, streamed as it lands. Latency, tokens, and cost broken down by model, team, and project without waiting on a nightly rollup.",
  },
  {
    icon: ShieldCheck,
    title: "Govern",
    description:
      "Write policy once and apply it at the gateway. Test against real traffic, then promote from draft to enforced with a full change record.",
  },
  {
    icon: CircleDollarSign,
    title: "Control spend",
    description:
      "Hard budgets per team and per project. Overruns alert first, throttle second, and stop third — thresholds you choose, not invoices you discover.",
  },
  {
    icon: ScrollText,
    title: "Prove it",
    description:
      "Export the audit trail on demand. Seven years of retention, immutable, and shaped for the auditor who asks the same question twice.",
  },
] as const;

export const MARKETING_SECURITY = [
  {
    icon: Shield,
    title: "SOC 2 Type II",
    description: "Audited annually, with reports available under NDA.",
  },
  {
    icon: Lock,
    title: "Data residency",
    description: "Pin processing to EU or US regions per workspace.",
  },
  {
    icon: KeyRound,
    title: "SSO and SCIM",
    description: "SAML SSO with automated deprovisioning on every plan.",
  },
  {
    icon: Server,
    title: "No training on your data",
    description: "Prompts are never used to train models, ours or anyone else's.",
  },
] as const;

export const MARKETING_TESTIMONIALS = [
  {
    quote:
      "We found four hundred thousand dollars a quarter of shadow LLM spend in the first week. The attribution view paid for the contract.",
    name: "Priya Raghunathan",
    role: "VP Engineering",
    company: "Northwind Logistics",
  },
  {
    quote:
      "Security signed off in one meeting because the audit trail already answered every question they had. That has never happened before.",
    name: "Marcus Oyelaran",
    role: "Head of Platform",
    company: "Halden Financial",
  },
  {
    quote:
      "Our engineers did not have to change a line of code. We moved three hundred services onto the gateway over a long weekend.",
    name: "Elena Vasquez",
    role: "Director of Infrastructure",
    company: "Corvus Health",
  },
] as const;

export const MARKETING_PERSONAS = [
  {
    icon: Code2,
    title: "Platform engineers",
    description:
      "Swap models and providers behind one endpoint, and stop owning per-vendor failover logic.",
  },
  {
    icon: Shield,
    title: "Security and compliance",
    description:
      "Enforce residency and redaction rules on every request, and export evidence when an auditor asks.",
  },
  {
    icon: CircleDollarSign,
    title: "Finance",
    description:
      "Get a per-team number that matches the finance system, without asking engineering for a spreadsheet.",
  },
  {
    icon: Users,
    title: "Engineering leadership",
    description:
      "See adoption and spend on one board, and know which pilots deserve to become products.",
  },
] as const;
