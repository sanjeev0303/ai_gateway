import type { Metadata } from "next";

import { CallToAction } from "@/components/marketing/cta";
import { FeatureGrid } from "@/components/marketing/feature-grid";
import { Hero } from "@/components/marketing/hero";
import { PersonaGrid } from "@/components/marketing/persona-grid";
import { APP_NAME } from "@/lib/constants";

export const metadata: Metadata = {
  title: `Solutions · ${APP_NAME}`,
  description:
    "How platform, security, and finance teams each use Sentinel to answer a different question about AI traffic.",
};

export default function SolutionsPage() {
  return (
    <>
      <Hero
        headline="Three teams, one source of truth"
        subheadline="Sentinel answers the question each of your stakeholders actually needs answered about AI usage — without building three separate systems."
      />
      <FeatureGrid
        eyebrow="By team"
        heading="Built around the questions you get asked"
        subheading="The same request log serves engineering, security, and finance, so nobody is reconciling screenshots."
      />
      <PersonaGrid
        eyebrow="By team"
        heading="The question behind every purchase"
        subheading="Platform, security, finance, and leadership each start with a different question. The request log answers all four."
      />
      <CallToAction
        heading="Give every team the same answer"
        body="Start with one project, invite your security and finance reviewers, and let the audit log build itself."
        secondaryLabel="See how it works"
        secondaryHref="/product"
      />
    </>
  );
}
