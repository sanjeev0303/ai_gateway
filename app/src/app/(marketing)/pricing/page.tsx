import type { Metadata } from "next";

import { CallToAction } from "@/components/marketing/cta";
import { Hero } from "@/components/marketing/hero";
import { PricingTable } from "@/components/marketing/pricing-table";
import { APP_NAME } from "@/lib/constants";

export const metadata: Metadata = {
  title: `Pricing · ${APP_NAME}`,
  description:
    "Sentinel is pre-release and not yet priced. What the platform does today, without a made-up price list.",
};

export default function PricingPage() {
  return (
    <>
      <Hero
        headline="Not priced yet"
        subheadline="There is no billing, invoicing, or seat management in the product yet, so any price list here would be fiction. Here is what the platform actually does today."
      />
      <PricingTable />
      <CallToAction
        heading="Send one request through it"
        body="The fastest way to understand the control plane is to route a single request through the gateway and inspect the record it writes."
        primaryLabel="Open the dashboard"
        secondaryLabel="Learn what you get"
        secondaryHref="/product"
      />
    </>
  );
}
