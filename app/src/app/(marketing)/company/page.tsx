import type { Metadata } from "next";
import { Eye, Gauge, Lock, Scale } from "lucide-react";

import { CallToAction } from "@/components/marketing/cta";
import { Hero } from "@/components/marketing/hero";
import { APP_NAME, MARKETING_NAV } from "@/lib/constants";

export const metadata: Metadata = {
  title: `Company · ${APP_NAME}`,
  description:
    "Sentinel exists so that enterprise AI usage can be explained, attributed, and audited without slowing anyone down.",
};

const PRINCIPLES = [
  {
    icon: Eye,
    title: "Observable by default",
    description:
      "If a request can reach a model, it can be attributed to a person and a project. No sampling, no blind spots.",
  },
  {
    icon: Lock,
    title: "Deny by default",
    description:
      "A credential resolves to exactly one tenant, and an immutable policy snapshot is consulted before the provider is called.",
  },
  {
    icon: Gauge,
    title: "Cheap enough to keep",
    description:
      "Authorization, rate limits, and budget checks sit on the hot path, so they read a compiled snapshot instead of querying on every request.",
  },
  {
    icon: Scale,
    title: "Explainable after the fact",
    description:
      "Requests keep their model, tokens, and latency. Tenant, credential, and policy changes are recorded and exportable on demand.",
  },
] as const;

export default function CompanyPage() {
  return (
    <>
      <Hero
        headline="AI adoption outran AI accountability"
        subheadline="Teams adopted model providers faster than they adopted controls. Sentinel is the layer that closes the gap, without becoming a bottleneck in the process."
      />

      <section className="border-b">
        <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6">
          <div className="max-w-2xl">
            <p className="text-sm font-semibold uppercase tracking-wide text-primary">
              How we build
            </p>
            <h2 className="mt-3 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
              Four principles we do not trade away
            </h2>
            <p className="mt-4 text-lg text-muted-foreground text-pretty">
              Every request to add a feature gets checked against these. If a
              change makes the gateway slower, less observable, or less
              explainable, it does not ship.
            </p>
          </div>

          <ul className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {PRINCIPLES.map((principle) => {
              const Icon = principle.icon;
              return (
                <li key={principle.title} className="rounded-xl border bg-card p-6">
                  <span className="inline-flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Icon className="size-5" aria-hidden />
                  </span>
                  <h3 className="mt-4 font-semibold">{principle.title}</h3>
                  <p className="mt-2 text-sm text-muted-foreground text-pretty">
                    {principle.description}
                  </p>
                </li>
              );
            })}
          </ul>
        </div>
      </section>

      <section className="border-b">
        <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6">
          <div className="max-w-2xl">
            <h2 className="text-3xl font-semibold tracking-tight text-balance">
              No sample data
            </h2>
            <p className="mt-4 text-lg text-muted-foreground text-pretty">
              There is no demo workspace behind these pages. Every panel in the
              dashboard is a live query against the control API, scoped to the
              tenant you are signed in as, and a failed query says so instead of
              rendering a placeholder.
            </p>
          </div>
        </div>
      </section>

      <CallToAction
        heading="Put the control plane in front of your traffic"
        body="Create a tenant, issue a scoped credential, and route your first request through the gateway. Invite your security reviewer when the first audit record lands."
        secondaryLabel={`Read the ${MARKETING_NAV[0]?.label.toLowerCase() ?? "product"} overview`}
        secondaryHref="/product"
      />
    </>
  );
}
