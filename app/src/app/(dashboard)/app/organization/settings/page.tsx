import type { Metadata } from "next";

import { PageShell } from "@/components/dashboard";
import { OrgSettings } from "@/components/dashboard/org-settings";
import { organization } from "@/lib/mock-data";

export const metadata: Metadata = {
  title: "Settings · Sentinel",
  description: "Organization name, seats, security defaults, and notification routing.",
};

export default function OrganizationSettingsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Settings"
        description="Organization-wide defaults. Per-project overrides live in each project."
      >
        <OrgSettings org={organization} />
      </PageShell>
    </div>
  );
}
