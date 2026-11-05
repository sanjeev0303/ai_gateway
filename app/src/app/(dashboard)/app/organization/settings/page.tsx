import type { Metadata } from "next";

import { PageShell } from "@/components/dashboard";
import { OrgSettings } from "@/components/dashboard/org-settings";
import { getCurrentOrganization } from "@/lib/session";

export const metadata: Metadata = {
  title: "Settings · Tollseal",
  description: "View organization identity and status; edit the name and slug.",
};

export default async function OrganizationSettingsPage() {
  const organization = await getCurrentOrganization();
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Settings"
        description="Organization-wide defaults. Name and slug are the only writable fields."
      >
        <OrgSettings org={organization} />
      </PageShell>
    </div>
  );
}
