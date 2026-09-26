import type { Metadata } from "next";
import Link from "next/link";
import type { ComponentType, CSSProperties, SVGProps } from "react";

import {
  ArrowRightIcon,
  CheckIcon,
  GaugeIcon,
  LayersIcon,
  PlugIcon,
  ScrollIcon,
  ShieldIcon,
  TerminalIcon,
} from "@/components/icons";
import { controls, deployment, hosted } from "@/lib/pricing";
import { site } from "@/lib/site";

export const metadata: Metadata = {
  title: "Govern enterprise AI usage",
};

type Icon = ComponentType<SVGProps<SVGSVGElement>>;

type Feature = {
  readonly icon: Icon;
  readonly title: string;
  readonly body: string;
  readonly meta: string;
};

const features: readonly Feature[] = [
  {
    icon: LayersIcon,
    title: "Scoped access for every caller",
    body: "Organizations, teams, projects, and users each get scoped credentials. A service credential can only do what its scope says it can.",
    meta: "org / team / project / user",
  },
  {
    icon: ShieldIcon,
    title: "Compiled policy snapshots",
    body: "Policies compile into immutable snapshots loaded ahead of the request. Authorization is a lookup, not a live evaluation.",
    meta: "compiled · immutable",
  },
  {
    icon: GaugeIcon,
    title: "Rate limits and budgets in the hot path",
    body: "Per-credential rate limits and monthly budgets are checked before the request is forwarded. Over budget fails closed.",
    meta: "checked before forward",
  },
  {
    icon: PlugIcon,
    title: "Provider-native passthrough",
    body: "Anthropic, OpenAI, and Gemini payloads pass through unchanged, including Claude Code. No universal schema forced on the wire.",
    meta: "anthropic · openai · gemini",
  },
  {
    icon: ScrollIcon,
    title: "Idempotent usage schema",
    body: "Usage events carry a unit and quantity, keyed unique per organization, request, and event type so replayed writes cannot double count. Not wired up yet.",
    meta: "idempotent ingest · not wired up",
  },
  {
    icon: TerminalIcon,
    title: "Self-serve control plane",
    body: "Manage tenants, policies, credentials, and audit from the web app or the REST API. OpenAPI generated, contracts shared.",
    meta: "web + REST · /v1",
  },
];

type Process = {
  readonly name: string;
  readonly port?: string;
  readonly role: string;
};

const processes: readonly Process[] = [
  {
    name: "gateway",
    port: ":8080",
    role: "Data plane hot path: auth, policy snapshot, rate and budget checks, forward, stream.",
  },
  {
    name: "gateway-v2",
    port: ":8084",
    role: "Next-generation proxy, currently serving provider passthrough traffic.",
  },
  {
    name: "control-api",
    port: ":8081",
    role: "Control plane: tenants, policies, credentials, audit, and billing over REST.",
  },
  {
    name: "worker",
    port: ":8082",
    role: "Async worker: health probes only. Rollups, retention, and alerting are not wired up.",
  },
  {
    name: "migrate · seed",
    role: "One-shot processes that apply migrations and optionally load demo data.",
  },
  {
    name: "web",
    port: ":3000",
    role: "Next.js dashboard and this site, talking to the control API.",
  },
];

const docs = [
  {
    tag: "Start here",
    title: "Getting started",
    body: "Run the full stack with Docker Compose, then create an organization, add a provider, and issue a first governed request.",
    href: "/docs/getting-started",
  },
  {
    tag: "Reference",
    title: "Architecture",
    body: "Data plane and control plane separation, the hot path order, and the repo layout across backend, apps, and packages.",
    href: "/docs/architecture",
  },
  {
    tag: "API",
    title: "Control plane API",
    body: "REST endpoints under /v1, authentication with JWTs and service credentials, idempotency, and the shared OpenAPI contract.",
    href: "/docs/api",
  },
  {
    tag: "Ops",
    title: "Deployment",
    body: "Docker Compose, Kubernetes, and Terraform targets, environment variables, health probes, and upgrade notes.",
    href: "/docs/deployment",
  },
] as const;

function Check({ size = 16 }: { readonly size?: number }) {
  return <CheckIcon width={size} height={size} />;
}

export default function MarketingHome() {
  return (
    <>
      {/* hero */}
      <section className="hero">
        <div className="l-wrap hero-inner">
          <div>
            <h1 className="hero-title">
              Every AI request, <em>governed</em> before it reaches a provider
            </h1>
            <p className="hero-lead">
              {site.name} sits between your applications and Anthropic, OpenAI,
              and Gemini. Credentials are scoped, policies are compiled into
              immutable snapshots, and every request is budgeted, rate limited,
              and audited.
            </p>
            <div className="hero-ctas">
              <Link href="/signup" className="btn btn-primary">
                Get started
              </Link>
              <Link href="/docs/getting-started" className="btn btn-secondary">
                Read the docs
              </Link>
            </div>
            <p className="hero-note">
              Self-hostable in one stack: <span className="lit">docker compose up</span>
            </p>
          </div>

          <div className="term" aria-label="Example of a governed request through the gateway">
            <div className="term-bar">
              <span className="term-dot" />
              <span className="term-dot" />
              <span className="term-dot" />
              <span className="term-title">gateway :8080 · request trace</span>
            </div>
            <div className="term-body">
              <p className="term-line">$ curl -sS gateway:8080/v1/messages \</p>
              <p className="term-line">  -H "authorization: Bearer aigw_live_9f2..." \</p>
              <p className="term-line">  -d {'{"model":"claude-sonnet-4-5","stream":true}'}</p>
              <div className="term-gap" />
              <div className="term-row term-step" style={{ "--d": "420ms" } as CSSProperties}>
                <span className="term-line">→ authn   aigw_live_9f2… · project payments</span>
                <span className="term-line term-line-ok term-status">PASS</span>
              </div>
              <div className="term-row term-step" style={{ "--d": "520ms" } as CSSProperties}>
                <span className="term-line">→ policy  snapshot v41 · rules 12 · immutable</span>
                <span className="term-line term-line-ok term-status">PASS</span>
              </div>
              <div className="term-row term-step" style={{ "--d": "620ms" } as CSSProperties}>
                <span className="term-line">→ rate    60 req/min · 47 used · 13 left</span>
                <span className="term-line term-line-ok term-status">PASS</span>
              </div>
              <div className="term-row term-step" style={{ "--d": "720ms" } as CSSProperties}>
                <span className="term-line">→ budget  $2,400.00/mo · $1,712.30 used</span>
                <span className="term-line term-line-ok term-status">PASS</span>
              </div>
              <div className="term-row term-step" style={{ "--d": "820ms" } as CSSProperties}>
                <span className="term-line">→ forward anthropic · claude-sonnet-4-5 · stream</span>
                <span className="term-line term-line-warn term-status">OK</span>
              </div>
              <div className="term-gap" />
              <p className="term-line term-line-dim term-step" style={{ "--d": "920ms" } as CSSProperties}>
                usage  input 1,204 · cached 3,108 · output 942 · thinking 0 · tool 84
              </p>
              <p className="term-line term-line-dim term-step" style={{ "--d": "990ms" } as CSSProperties}>
                audit  written · rollup hourly · idempotent per request_id
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* product */}
      <section id="product" className="features l-section">
        <div className="l-wrap">
          <div className="section-head">
            <h2 className="section-title">Governance lives in the hot path</h2>
            <p className="section-lead">
              Authorization, rate limits, and budget checks run at request time,
              backed by compiled, immutable policy snapshots. The data plane
              does not call home for permission.
            </p>
          </div>
          <div className="l-grid features-grid">
            {features.map((f) => (
              <article key={f.title} className="feature-card">
                <div className="feature-icon">
                  <f.icon width={20} height={20} />
                </div>
                <h3 className="feature-title">{f.title}</h3>
                <p className="feature-body">{f.body}</p>
                <span className="feature-meta">{f.meta}</span>
              </article>
            ))}
          </div>
        </div>
      </section>

      {/* architecture */}
      <section id="architecture" className="l-section">
        <div className="l-wrap">
          <div className="section-head">
            <h2 className="section-title">A data plane and a control plane</h2>
            <p className="section-lead">
              Six Go processes and one web app, in a single monorepo. Data
              plane processes carry traffic; control plane processes carry
              policy, credentials, and audit.
            </p>
          </div>
          <div className="l-grid arch-grid">
            {processes.map((p) => (
              <article key={p.name} className="arch-card">
                <h3 className="arch-name">
                  {p.name}
                  {p.port ? <span className="lit">{p.port}</span> : null}
                </h3>
                <p className="arch-role">{p.role}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      {/* pricing */}
      <section id="pricing" className="pricing l-section">
        <div className="l-wrap">
          <div className="section-head">
            <h2 className="section-title">What the gateway actually enforces</h2>
            <p className="section-lead">
              Most pricing pages list seat counts. These are the controls the
              data plane evaluates on every request, with the exact values the
              code uses. Each row names the type or function it comes from, so
              you can check it against the repository.
            </p>
          </div>

          <div className="controls">
            {controls.map((c) => (
              <div key={c.name} className="control-row">
                <div className="control-what">
                  <p className="control-name">{c.name}</p>
                  <p className="control-summary">{c.summary}</p>
                </div>
                <ul className="control-values">
                  {c.values.map((v) => (
                    <li key={v} className="control-value">
                      {v}
                    </li>
                  ))}
                </ul>
                <p className="control-source">{c.source}</p>
              </div>
            ))}
          </div>

          <div className="plans">
            <article className="plan">
              <div className="plan-head">
                <p className="plan-name">{hosted.name}</p>
                <p className="plan-price plan-price-na">{hosted.price}</p>
              </div>
              <p className="plan-body">{hosted.summary}</p>
              <Link href={hosted.href} className="btn btn-secondary plan-cta">
                {hosted.cta}
                <ArrowRightIcon width={14} height={14} />
              </Link>
            </article>

            <article className="plan plan-primary">
              <div className="plan-head">
                <p className="plan-name">{deployment.name}</p>
                <p className="plan-price">{deployment.price}</p>
              </div>
              <p className="plan-body">{deployment.summary}</p>
              <ul className="plan-points">
                {deployment.points.map((p) => (
                  <li key={p} className="plan-point">
                    <Check />
                    <span>{p}</span>
                  </li>
                ))}
              </ul>
              <Link href={deployment.href} className="btn btn-primary plan-cta">
                {deployment.cta}
                <ArrowRightIcon width={14} height={14} />
              </Link>
            </article>
          </div>
        </div>
      </section>

      {/* docs */}
      <section className="l-section">
        <div className="l-wrap">
          <div className="section-head">
            <h2 className="section-title">Docs grounded in the repository</h2>
            <p className="section-lead">
              Architecture decisions live in ADRs, the roadmap in milestones,
              and the working system in the repo. These pages point at what is
              real.
            </p>
          </div>
          <div className="l-grid docs-grid">
            {docs.map((d) => (
              <article key={d.title} className="doc-card">
                <span className="doc-card-tag">{d.tag}</span>
                <h3 className="doc-card-title">{d.title}</h3>
                <p className="doc-card-body">{d.body}</p>
                <Link href={d.href} className="doc-card-go">
                  Read the docs
                  <ArrowRightIcon width={14} height={14} />
                </Link>
              </article>
            ))}
          </div>
        </div>
      </section>

      {/* cta */}
      <section className="cta-band">
        <div className="l-wrap">
          <h2 className="cta-title">Route your AI traffic through one gateway</h2>
          <p className="cta-lead">
            Create an organization, attach a provider, and issue your first
            governed request in minutes.
          </p>
          <div className="cta-actions">
            <Link href="/signup" className="btn btn-primary">
              Get started free
            </Link>
            <Link href="/docs" className="btn btn-secondary">
              Explore the docs
            </Link>
          </div>
        </div>
      </section>
    </>
  );
}