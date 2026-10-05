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
      "Route provider traffic through POST /v1/messages. The request shape is Anthropic's, so existing Anthropic clients need no rewriting.",
  },
  {
    icon: BookOpen,
    title: "Credentials and policy",
    description:
      "The credential scopes the gateway enforces, and what an immutable policy snapshot does at request time. The snapshot is compiled from stored policy records; there is no API for authoring it yet.",
  },
  {
    icon: FileText,
    title: "Audit export",
    description:
      "Which tenant, credential, and policy changes get recorded, and how to export the trail. Retention windows are not implemented yet.",
  },
  {
    icon: LifeBuoy,
    title: "Client packages",
    description:
      "The private workspace packages that talk to the control API, plus how to call the REST API directly if you would rather not take a dependency.",
  },
] as const;

export default function ResourcesPage() {
  return (
    <>
      <Hero
        headline="Everything you need to run the gateway"
        subheadline="Quickstart, API reference, credential scopes, and client packages — for the parts that are real, and an explicit note where they are not."
      />

      <section className="border-b">
        <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6">
          <h2 className="text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            Start here
          </h2>
          <p className="mt-4 max-w-2xl text-lg text-muted-foreground text-pretty">
            These four references are all in the developer section of the
            dashboard. Each one states which backend capabilities exist today,
            because two of the four cover behaviour that is specified but not
            yet implemented.
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
