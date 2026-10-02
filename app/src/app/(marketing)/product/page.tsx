import type { Metadata } from "next";

import { CallToAction } from "@/components/marketing/cta";
import { FeatureGrid } from "@/components/marketing/feature-grid";
import { Hero } from "@/components/marketing/hero";
import { ProductShowcase } from "@/components/marketing/product-showcase";
import { SecuritySection } from "@/components/marketing/security-section";
import { APP_NAME } from "@/lib/constants";

export const metadata: Metadata = {
  title: `Product · ${APP_NAME}`,
  description:
    "A single gateway over every model provider, with policy enforcement, cost attribution, and an immutable audit trail.",
};

export default function ProductPage() {
  return (
    <>
      <Hero
        headline="Inspect, attribute, and record every request"
        subheadline="Sentinel proxies your AI traffic through a policy engine, tags every call with the user, team, and project that caused it, and writes an immutable record you can hand to an auditor."
      />
      <FeatureGrid
        eyebrow="Capabilities"
        heading="Everything the control plane does"
        subheading="Each capability works on its own, and gets stronger when you combine them."
      />
      <ProductShowcase />
      <SecuritySection />
      <CallToAction
        heading="Send your first request through the gateway"
        body="Swap one base URL, keep the same SDK, and watch attribution appear in the dashboard."
        primaryLabel="Open the dashboard"
        primaryHref="/app/overview"
        secondaryLabel="Compare plans"
        secondaryHref="/pricing"
      />
    </>
  );
}
