import type { Metadata } from "next";

import { CallToAction } from "@/components/marketing/cta";
import { Hero } from "@/components/marketing/hero";
import { PricingTable } from "@/components/marketing/pricing-table";
import { APP_NAME } from "@/lib/constants";

export const metadata: Metadata = {
  title: `Pricing · ${APP_NAME}`,
  description:
    "Start free with the gateway, then pay for policy enforcement, retention, and controls as your AI usage grows.",
};

export default function PricingPage() {
  return (
    <>
      <Hero
        headline="Start on the gateway. Pay for control."
        subheadline="No per-seat pricing theatre. Every plan includes the routing layer; you pay more when you need policy enforcement, long retention, and procurement-grade controls."
      />
      <PricingTable />
      <CallToAction
        heading="Try it against real traffic for a week"
        body="The Starter plan is free for five people. No card, no sales call, no procurement cycle."
        primaryLabel="Open the dashboard"
        secondaryLabel="Learn what you get"
        secondaryHref="/product"
      />
    </>
  );
}
