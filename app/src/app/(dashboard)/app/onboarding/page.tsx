import type { Metadata } from "next";
import { PageShell } from "@/components/dashboard";
import { getCurrentOrganization, getApiClient, requireSession } from "@/lib/session";
import { OnboardingClient } from "./onboarding-client";

export const metadata: Metadata = {
  title: "Employee Onboarding · Sentinel",
  description: "Guided onboarding: verify access, obtain credentials, and run test requests.",
};

export default async function EmployeeOnboardingPage() {
  const session = await requireSession();
  const org = await getCurrentOrganization();
  const client = await getApiClient();

  const [teamsRes, projectsRes, modelsRes, credsRes] = await Promise.all([
    client.teams.list().catch(() => ({ items: [] })),
    client.projects.list().catch(() => ({ items: [] })),
    client.models.list().catch(() => ({ items: [] })),
    client.credentials.listGateway().catch(() => ({ items: [] })),
  ]);

  return (
    <div className="p-4 sm:p-6 max-w-5xl mx-auto">
      <PageShell
        title="Employee Onboarding"
        description="Verify your organization membership, inspect authorized models, issue your gateway key, and test connectivity."
      >
        <OnboardingClient
          user={{
            id: session.claims.subject,
            name: session.claims.display_name,
            email: session.claims.email,
            role: session.claims.role,
          }}
          organization={org}
          teams={teamsRes.items || []}
          projects={projectsRes.items || []}
          models={modelsRes.items || []}
          existingCredentials={credsRes.items || []}
        />
      </PageShell>
    </div>
  );
}
