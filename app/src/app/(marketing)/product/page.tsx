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
    "A single gateway over every model provider, with cost attribution and a recorded request trail.",
};

export default function ProductPage() {
  return (
    <>
      <Hero
        headline="Inspect, attribute, and record every request"
        subheadline="Sentinel proxies your AI traffic through one gateway, tags every call with the user, team, and project that caused it, and writes the cost and outcome to a recorded request trail."
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
