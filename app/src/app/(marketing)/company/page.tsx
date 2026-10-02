import type { Metadata } from "next";
import { Eye, Gauge, Lock, Scale } from "lucide-react";

import { CallToAction } from "@/components/marketing/cta";
import { Hero } from "@/components/marketing/hero";
import { Testimonials } from "@/components/marketing/testimonials";
import { APP_NAME, MARKETING_NAV } from "@/lib/constants";
import { organization } from "@/lib/mock-data";

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
    title: "Refuse by default",
    description:
      "Traffic is inspected before it is billed. A policy that blocks a request is a cheaper outcome than a leaked one.",
  },
  {
    icon: Gauge,
    title: "Fast enough to keep",
    description:
      "Enforcement runs in under 40ms. A security control nobody tolerates in the hot path is not a control.",
  },
  {
    icon: Scale,
    title: "Explainable after the fact",
    description:
      "Seven years of immutable records, exportable without asking us, because the auditor should not need a favour.",
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
          <div className="grid gap-10 md:grid-cols-2">
            <div>
              <h2 className="text-3xl font-semibold tracking-tight text-balance">
                Where this runs today
              </h2>
              <p className="mt-4 text-lg text-muted-foreground text-pretty">
                The dashboard is populated from a representative Acme Corp
                Analytics workspace, so every figure you see is a real record
                with a real policy decision attached to it.
              </p>
            </div>
            <dl className="grid grid-cols-2 gap-6 self-start">
              <div className="rounded-lg border bg-card p-4">
                <dt className="text-sm text-muted-foreground">Organization</dt>
                <dd className="mt-1 font-medium">{organization.name}</dd>
              </div>
              <div className="rounded-lg border bg-card p-4">
                <dt className="text-sm text-muted-foreground">Region</dt>
                <dd className="mt-1 font-medium">{organization.region}</dd>
              </div>
              <div className="rounded-lg border bg-card p-4">
                <dt className="text-sm text-muted-foreground">Seats in use</dt>
                <dd className="mt-1 font-medium tabular-nums">
                  {organization.seatsUsed} of {organization.seatsTotal}
                </dd>
              </div>
              <div className="rounded-lg border bg-card p-4">
                <dt className="text-sm text-muted-foreground">Billing contact</dt>
                <dd className="mt-1 truncate font-medium">
                  {organization.billingEmail}
                </dd>
              </div>
            </dl>
          </div>
        </div>
      </section>

      <Testimonials />
      <CallToAction
        heading="Put the control plane in front of your traffic"
        body="Start with one project on the free plan. Invite your security reviewer when the first audit record lands."
        secondaryLabel={`Read the ${MARKETING_NAV[0]?.label.toLowerCase() ?? "product"} overview`}
        secondaryHref="/product"
      />
    </>
  );
}
