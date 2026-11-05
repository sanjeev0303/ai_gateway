import Link from "next/link";
import {
  AlertTriangle,
  ArrowRight,
  Check,
  Layers,
  Receipt,
  Route,
  ScrollText,
  ShieldCheck,
  Wallet,
} from "lucide-react";

import { FlowCurves } from "@/components/brand/flow-curves";
import { GuillocheBand, GuillocheRosette } from "@/components/brand/guilloche";
import { LogoMark } from "@/components/logo";
import { CodeTabs } from "./code-tabs";
import { Container, SectionIntro } from "./section";

/* ------------------------------------------------------------------------ */
/* Providers                                                                 */
/* ------------------------------------------------------------------------ */

const PROVIDERS = ["OpenAI", "Anthropic", "Google Gemini", "Azure OpenAI", "Mistral", "Hugging Face"];

export function ProviderStrip() {
  return (
    <section aria-label="Providers Tollseal routes to" className="border-y border-border">
      <Container className="flex flex-col gap-4 py-7 md:flex-row md:items-center md:gap-10">
        <p className="shrink-0 font-mono text-xs tracking-[0.08em] text-muted-foreground">Routes to</p>
        <ul className="flex flex-1 flex-wrap items-center justify-between gap-x-8 gap-y-3">
          {PROVIDERS.map((name) => (
            <li key={name} className="font-display text-[1rem] text-foreground/75">
              {name}
            </li>
          ))}
          <li className="text-sm text-muted-foreground">and any OpenAI-compatible endpoint</li>
        </ul>
      </Container>
    </section>
  );
}

/* ------------------------------------------------------------------------ */
/* The problem, in fixed tiers: the risk, what happens, what Tollseal does    */
/* ------------------------------------------------------------------------ */

const PROBLEMS = [
  {
    risk: "A different key, SDK and format per provider",
    happens: "Switching models means new credentials and rewritten code, and every team wires up providers its own way.",
    fix: "Connect each provider once. Everyone calls one OpenAI- or Anthropic-style endpoint, so changing models is a one-word change.",
  },
  {
    risk: "Provider keys live in every app",
    happens: "One leaked key exposes a whole provider account, and nobody can say which service it came from.",
    fix: "Provider keys stay with the gateway. People and services get aigw_ keys scoped to a team, project or service account, each revocable on its own.",
  },
  {
    risk: "Prompts carry personal data and secrets",
    happens: "Emails, phone numbers and pasted API keys go straight to a third-party model.",
    fix: "Guardrails block, redact or warn on PII, secrets and blocked keywords before a request leaves.",
  },
  {
    risk: "Spend that nobody can attribute",
    happens: "The invoice arrives as one number, with no team, project or person on it.",
    fix: "Every request is priced and tagged, with spend, token and request budgets per team or project.",
  },
  {
    risk: "No record of who asked what",
    happens: "When something goes wrong there is nothing to audit, and a different rule in every service.",
    fix: "One policy for every caller, and a record of every request: tokens, cost, latency and outcome.",
  },
] as const;

export function ProblemRegister() {
  return (
    <section aria-labelledby="problem-heading" className="py-24 lg:py-32">
      <Container className="flex flex-col gap-12">
        <SectionIntro
          id="problem-heading"
          title="Many providers, many keys, no single record."
          body="Every team reaches for a different model with a different key. These are the gaps that opens, and what closes each one."
        />
        <div role="table" aria-label="The problems Tollseal solves" className="overflow-hidden rounded-2xl border border-border-strong bg-card">
          <div role="row" className="hidden grid-cols-[1fr_1.3fr_1.3fr] gap-6 border-b border-border-strong bg-background/60 px-6 py-3 text-xs font-medium text-muted-foreground md:grid">
            <span role="columnheader">The risk</span>
            <span role="columnheader">What happens</span>
            <span role="columnheader">What Tollseal does</span>
          </div>
          {PROBLEMS.map((p) => (
            <div
              key={p.risk}
              role="row"
              className="grid grid-cols-1 gap-3 border-b border-border px-6 py-5 last:border-b-0 md:grid-cols-[1fr_1.3fr_1.3fr] md:gap-6"
            >
              <p role="cell" className="flex items-start gap-2.5 text-[0.9375rem] font-semibold text-foreground">
                <AlertTriangle className="mt-0.5 size-4 shrink-0 text-rose-600 dark:text-rose-400" aria-hidden />
                {p.risk}
              </p>
              <p role="cell" className="text-sm leading-relaxed text-muted-foreground">
                <span className="mb-1 block text-xs font-medium text-muted-foreground md:hidden">What happens</span>
                {p.happens}
              </p>
              <p role="cell" className="text-sm leading-relaxed text-foreground">
                <span className="mb-1 block text-xs font-medium text-muted-foreground md:hidden">What Tollseal does</span>
                {p.fix}
              </p>
            </div>
          ))}
        </div>
      </Container>
    </section>
  );
}

/* ------------------------------------------------------------------------ */
/* Capabilities, proven by the real console                                  */
/* ------------------------------------------------------------------------ */

const CAPABILITIES = [
  { icon: Route, title: "AI gateway", body: "One endpoint, OpenAI and Anthropic request formats, every provider behind it." },
  { icon: Layers, title: "Model access", body: "Allow or block models for the whole organization, a team or a project." },
  { icon: ShieldCheck, title: "Policies and guardrails", body: "Block, redact or warn on PII, secrets and keywords before a prompt leaves." },
  { icon: Receipt, title: "Usage and cost", body: "Tokens and spend by team, project, model and person." },
  { icon: Wallet, title: "Budgets and limits", body: "Spend, token and request caps on an hourly, daily, weekly or monthly window." },
  { icon: ScrollText, title: "Audit trail", body: "Who changed what and when, ready to export." },
];

/* A fragment of the teams page, drawn in HTML. Sample figures, labelled so. */
const SAMPLE_TEAMS = [
  { team: "Engineering", spend: 412.8 },
  { team: "Data Science", spend: 268.15 },
  { team: "Support", spend: 96.4 },
  { team: "Security", spend: 54.1 },
  { team: "Finance", spend: 21.75 },
  { team: "Marketing", spend: 12.3 },
];
const SAMPLE_FIGURES = [
  { label: "Requests", value: "48,210" },
  { label: "Tokens", value: "31.6M" },
  { label: "Spend", value: "$865.50" },
];

function ConsoleFragment() {
  const top = SAMPLE_TEAMS[0]!.spend;
  return (
    <figure
      aria-label="A fragment of the Tollseal teams page with sample figures: spend ranked by team"
      className="overflow-hidden rounded-2xl border border-border-strong bg-card shadow-(--shadow-dialog)"
    >
      <div className="flex items-center justify-between gap-3 border-b border-border px-5 py-3.5">
        <span className="text-[0.9375rem] font-semibold">Teams</span>
        <span className="rounded-[4px] border border-border-strong px-2 py-0.5 font-mono text-[0.625rem] tracking-[0.16em] text-muted-foreground">
          SAMPLE FIGURES
        </span>
      </div>
      <dl className="grid grid-cols-3 gap-px border-b border-border bg-border">
        {SAMPLE_FIGURES.map((f) => (
          <div key={f.label} className="flex flex-col gap-1 bg-card px-5 py-4">
            <dt className="text-xs text-muted-foreground">{f.label}</dt>
            <dd className="font-display text-[1.5rem] leading-none tabular-nums">{f.value}</dd>
          </div>
        ))}
      </dl>
      <div className="flex flex-col gap-3 px-5 py-5">
        <span className="text-sm font-semibold">Spend by team</span>
        <ol className="flex flex-col gap-2.5">
          {SAMPLE_TEAMS.map((t) => (
            <li key={t.team} className="grid grid-cols-[6.5rem_minmax(0,1fr)_4.5rem] items-center gap-3 text-xs">
              <span className="truncate text-muted-foreground">{t.team}</span>
              <span className="h-2 overflow-hidden rounded-full bg-muted" aria-hidden>
                <span className="block h-full rounded-full bg-primary" style={{ width: `${(t.spend / top) * 100}%` }} />
              </span>
              <span className="text-right font-mono tabular-nums">${t.spend.toFixed(2)}</span>
            </li>
          ))}
        </ol>
      </div>
    </figure>
  );
}

export function Capabilities() {
  return (
    <section id="solutions" aria-labelledby="capabilities-heading" className="scroll-mt-20 border-t border-border bg-card/60 py-24 lg:py-32">
      <Container className="grid grid-cols-1 items-center gap-14 lg:grid-cols-[minmax(0,0.75fr)_minmax(0,1.25fr)]">
        <div className="flex flex-col gap-10">
          <SectionIntro
            id="capabilities-heading"
            title="One console. One record."
            body="Every page reads from the same record of every request, so the numbers agree."
          />
          <ul className="flex flex-col divide-y divide-border border-y border-border">
            {CAPABILITIES.map(({ icon: Icon, title, body }) => (
              <li key={title} className="flex gap-3.5 py-3.5">
                <Icon className="mt-0.5 size-[18px] shrink-0 text-copper" aria-hidden />
                <span className="flex flex-col gap-0.5">
                  <span className="text-[0.9375rem] font-semibold">{title}</span>
                  <span className="text-sm leading-relaxed text-muted-foreground">{body}</span>
                </span>
              </li>
            ))}
          </ul>
        </div>

        <ConsoleFragment />
      </Container>
    </section>
  );
}

/* ------------------------------------------------------------------------ */
/* Security, and where it runs                                               */
/* ------------------------------------------------------------------------ */

const GUARDS = [
  { title: "Provider keys stay with the gateway", body: "Apps hold a scoped Tollseal key, so a leak never exposes a provider account." },
  { title: "Personal data masked", body: "Emails, phone numbers, SSNs and keys are blocked or masked before forwarding." },
  { title: "Model restrictions", body: "Allowlists and blocklists by organization, team, project, key or service account." },
  { title: "Audit logs", body: "Every console change, who made it and when, ready to export." },
  { title: "Policy on every request", body: "Rules compile into a snapshot the gateway checks before it forwards." },
  { title: "Security events and alerts", body: "Provider failures and anomalies raise events and alert rules." },
];

export function Security() {
  return (
    <section id="trust" aria-labelledby="security-heading" className="scroll-mt-20 py-24 lg:py-32">
      <Container className="grid grid-cols-1 gap-12 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)] lg:gap-16">
        <SectionIntro
          id="security-heading"
          title="Built for security. Runs on your machines."
          body="A Go gateway, a control API and a worker, with Postgres, Redis and NATS, as containers you run: with Docker Compose on one machine, in your cloud VPC next to the services that call it, or on-premise when data cannot leave your network."
        />
        {/* A register of mechanisms in the product today, not a grid of feature cards */}
        <ul aria-label="Security mechanisms" className="divide-y divide-border border-y border-border-strong">
          {GUARDS.map(({ title, body }) => (
            <li key={title} className="grid grid-cols-[1.25rem_minmax(0,1fr)] gap-x-3 py-4 sm:grid-cols-[1.25rem_minmax(0,0.9fr)_minmax(0,1.1fr)] sm:gap-x-6">
              <Check className="mt-0.5 size-4 text-emerald-600 dark:text-emerald-400" aria-hidden />
              <span className="text-[0.9375rem] font-semibold">{title}</span>
              <span className="col-start-2 text-sm leading-relaxed text-muted-foreground sm:col-start-3">{body}</span>
            </li>
          ))}
        </ul>
      </Container>
    </section>
  );
}

/* ------------------------------------------------------------------------ */
/* Developers: the one band printed in intaglio ink                          */
/* ------------------------------------------------------------------------ */

const CONNECT_STEPS = [
  {
    title: "Connect your providers",
    body: "OpenAI, Anthropic, Google Gemini, Azure OpenAI, Hugging Face, Mistral and any OpenAI-compatible endpoint. Their keys stay with the gateway.",
    code: "Console · Platform · Provider accounts",
  },
  {
    title: "Issue gateway keys",
    body: "An aigw_ key for a person, team, project or service account. Rotate, expire or revoke one without touching the others.",
    code: "aigw credential create",
  },
  {
    title: "Call from anywhere",
    body: "The aigw CLI, the TypeScript SDK, or OpenAI- and Anthropic-style REST. One key reaches every model your policy allows.",
    code: "aigw request -m claude-sonnet-4-5",
  },
];

export function Developers() {
  return (
    <section id="integration" aria-labelledby="developers-heading" className="dark scroll-mt-20 bg-background py-24 text-foreground [color-scheme:dark] lg:py-32">
      <Container className="grid grid-cols-1 items-center gap-14 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
        <div className="flex flex-col gap-10">
          <SectionIntro
            id="developers-heading"
            title="Connect providers once. Hand out one key."
            body="Add your provider accounts, issue gateway keys, and everyone calls through the gateway: from the terminal, from code, or over plain HTTP."
          >
            <div className="pt-2">
              <Link
                href="/resources"
                className="group inline-flex h-11 items-center gap-2 rounded-full border border-border-strong px-6 text-sm font-medium transition-[background-color,transform] duration-150 ease-(--ease-out-strong) hover:bg-accent focus-visible:ring-3 focus-visible:ring-ring/40 focus-visible:outline-none active:scale-[0.97]"
              >
                Read the documentation
                <ArrowRight className="size-4 transition-transform duration-200 group-hover:translate-x-0.5" aria-hidden />
              </Link>
            </div>
          </SectionIntro>
          <ol className="flex flex-col divide-y divide-border border-y border-border">
            {CONNECT_STEPS.map(({ title, body, code }, i) => (
              <li key={title} className="grid grid-cols-[2rem_minmax(0,1fr)] gap-x-3 py-4">
                <span className="font-display text-lg leading-6 text-copper tabular-nums" aria-hidden>
                  {i + 1}
                </span>
                <span className="flex min-w-0 flex-col gap-1">
                  <span className="text-[0.9375rem] font-semibold">{title}</span>
                  <span className="text-sm leading-relaxed text-muted-foreground">{body}</span>
                  <code className="mt-1 w-fit max-w-full truncate rounded-md border border-border px-2 py-0.5 text-xs text-foreground/85">{code}</code>
                </span>
              </li>
            ))}
          </ol>
        </div>
        <div className="min-w-0">
          <CodeTabs />
        </div>
      </Container>
    </section>
  );
}

/* ------------------------------------------------------------------------ */
/* Closing: the seal, one action, and the honest price                       */
/* ------------------------------------------------------------------------ */

const START_STEPS = [
  { title: "Connect a provider", body: "Add one API key. It stays with the gateway." },
  { title: "Issue a gateway key", body: "Scoped to a person, team or project." },
  { title: "Call it from the CLI, SDK or REST", body: "Your next request is checked and sealed." },
];

export function FinalCta() {
  return (
    <section aria-labelledby="cta-heading" className="py-24 lg:py-32">
      <Container className="flex flex-col gap-6">
        {/* Centred close: the curves arrive from both sides and meet at the seal. */}
        <div className="relative overflow-hidden rounded-[20px] border border-border-strong bg-card">
          <GuillocheBand className="absolute inset-x-0 top-0 opacity-80" />
          <div className="relative flex flex-col items-center gap-6 px-6 pt-14 pb-10 text-center sm:px-12">
            <h2 id="cta-heading" className="max-w-[18ch] font-display text-[clamp(2rem,4vw,3.25rem)] leading-[1.02]">
              Every provider behind one key.
            </h2>
            <p className="max-w-[52ch] text-[1.0625rem] leading-relaxed text-muted-foreground">
              Connect a provider, issue a gateway key, and your next request is checked and sealed.
            </p>
            <Link
              href="/signup"
              className="group inline-flex h-12 items-center gap-2 rounded-full bg-primary px-8 text-[0.9375rem] font-medium text-primary-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.16),0_8px_18px_-8px_rgba(13,92,72,0.6)] transition-[background-color,transform] duration-150 ease-(--ease-out-strong) hover:bg-primary-hover focus-visible:ring-3 focus-visible:ring-ring/40 focus-visible:outline-none active:scale-[0.97]"
            >
              Get started
              <ArrowRight className="size-4 transition-transform duration-200 group-hover:translate-x-0.5" aria-hidden />
            </Link>
          </div>
          <div className="relative h-56">
            <FlowCurves variant="converge" />
            <div className="absolute top-1/2 left-1/2 grid size-48 -translate-x-1/2 -translate-y-1/2 place-items-center rounded-full bg-card">
              <GuillocheRosette className="absolute inset-0 size-full" />
              <LogoMark className="relative size-16" />
            </div>
          </div>
          <ol className="relative grid grid-cols-1 gap-px border-t border-border bg-border sm:grid-cols-3">
            {START_STEPS.map(({ title, body }) => (
              <li key={title} className="bg-card px-6 py-5">
                <span className="block text-[0.9375rem] font-semibold">{title}</span>
                <span className="block text-sm text-muted-foreground">{body}</span>
              </li>
            ))}
          </ol>
        </div>

        <div id="pricing" className="flex scroll-mt-20 flex-col gap-1 rounded-2xl border border-border bg-card px-6 py-5 sm:flex-row sm:items-center sm:justify-between sm:gap-6">
          <p className="text-sm">
            <span className="font-semibold">Not priced yet.</span>{" "}
            <span className="text-muted-foreground">
              There is no billing or seat management in the product, so any price list would be fiction.
            </span>
          </p>
          <Link href="/pricing" className="inline-flex shrink-0 items-center gap-1.5 text-sm font-semibold text-primary underline-offset-4 hover:underline">
            See what exists today
            <ArrowRight className="size-4" aria-hidden />
          </Link>
        </div>
      </Container>
    </section>
  );
}
