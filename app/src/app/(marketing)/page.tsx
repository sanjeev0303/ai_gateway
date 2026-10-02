import {
  Architecture,
  Attribution,
  CodeTabs,
  Comparison,
  FinalCta,
  GatewayFlow,
  Governance,
  Hero,
  Pillars,
  PricingTeaser,
  Problem,
  ProviderStrip,
  RequestInspector,
  Trust,
  UseCases,
} from "@/components/landing";

export default function HomePage() {
  return (
    <>
      <Hero />
      <ProviderStrip />
      <Problem />
      <GatewayFlow />
      <Pillars />
      <RequestInspector />
      <UseCases />
      <Attribution />
      <Governance />
      <CodeTabs />
      <Architecture />
      <Comparison />
      <Trust />
      <PricingTeaser />
      <FinalCta />
    </>
  );
}
