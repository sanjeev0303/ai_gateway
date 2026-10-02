import type { Metadata } from "next";

import { OverviewContent } from "./overview-content";

export const metadata: Metadata = {
  title: "Overview · Sentinel",
  description: "Workspace health, spend, and policy posture at a glance.",
};

export default function OverviewPage() {
  return <OverviewContent />;
}
