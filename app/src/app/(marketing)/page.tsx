import { HomeHero } from "@/components/home/hero";
import { HomeMotion } from "@/components/home/home-motion";
import { Workflow } from "@/components/home/workflow";
import {
  Capabilities,
  Developers,
  FinalCta,
  ProblemRegister,
  ProviderStrip,
  Security,
} from "@/components/home/sections";

export default function HomePage() {
  return (
    <>
      <HomeHero />
      <ProviderStrip />
      <ProblemRegister />
      <Workflow />
      <Developers />
      <Capabilities />
      <Security />
      <FinalCta />
      <HomeMotion />
    </>
  );
}
