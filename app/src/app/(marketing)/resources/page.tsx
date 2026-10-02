import type { Metadata } from "next";
import { BookOpen, FileText, LifeBuoy, Terminal } from "lucide-react";

import { CallToAction } from "@/components/marketing/cta";
import { Hero } from "@/components/marketing/hero";
import { APP_NAME } from "@/lib/constants";

export const metadata: Metadata = {
  title: `Resources · ${APP_NAME}`,
  description:
    "Integration guides, API reference, and changelog for the Sentinel AI control plane.",
};

const RESOURCES = [
  {
    icon: Terminal,
    title: "Gateway quickstart",
    description:
      "Point an existing OpenAI-compatible client at the gateway, keep your model names, and confirm attribution in the dashboard.",
  },
  {
    icon: BookOpen,
    title: "Policy reference",
    description:
      "Every rule type Sentinel evaluates at the edge, how they combine, and what happens when a request fails inspection.",
  },
  {
    icon: FileText,
    title: "Audit and retention",
    description:
      "The exact record written for each request, decision, and configuration change — and how to export it for review.",
  },
  {
    icon: LifeBuoy,
    title: "Migration guide",
    description:
      "Moving an existing multi-provider setup behind one endpoint without downtime, dual writes, or a flag day.",
  },
] as const;

export default function ResourcesPage() {
  return (
    <>
      <Hero
        headline="Everything you need to run the gateway"
        subheadline="Integration notes, the policy reference, and the audit format your security team will ask about — written for people who have to actually operate this."
      />

      <section className="border-b">
        <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6">
          <h2 className="text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            Start here
          </h2>
          <p className="mt-4 max-w-2xl text-lg text-muted-foreground text-pretty">
            Most teams ship the gateway behind a single environment variable and
            reconcile the numbers afterwards. These four references cover that
            path end to end.
          </p>

          <ul className="mt-12 grid gap-6 sm:grid-cols-2">
            {RESOURCES.map((resource) => {
              const Icon = resource.icon;
              return (
                <li
                  key={resource.title}
                  className="rounded-xl border bg-card p-6"
                >
                  <span className="inline-flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Icon className="size-5" aria-hidden />
                  </span>
                  <h3 className="mt-4 font-semibold">{resource.title}</h3>
                  <p className="mt-2 text-sm text-muted-foreground text-pretty">
                    {resource.description}
                  </p>
                </li>
              );
            })}
          </ul>
        </div>
      </section>

      <CallToAction
        heading="Skip the reading and try the gateway"
        body="The fastest way to understand the control plane is to send one request through it and inspect the record it writes."
        secondaryLabel="See the capabilities"
        secondaryHref="/product"
      />
    </>
  );
}
