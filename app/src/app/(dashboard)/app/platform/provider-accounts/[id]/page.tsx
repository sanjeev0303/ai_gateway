import { notFound } from "next/navigation";
import { getApiClient } from "@/lib/session";
import { PageShell } from "@/components/dashboard";
import { ProviderDetailClient } from "./provider-detail-client";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Provider Account Detail · Sentinel",
};

export default async function ProviderAccountDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  try {
    const api = await getApiClient();
    const [account, credentials, health, models, usageByProvider] = await Promise.all([
      api.providers.getAccount(id),
      api.providers.listCredentials(),
      api.providers.health().catch(() => ({ items: [] })),
      api.models.list().catch(() => ({ items: [] })),
      api.usage.byProvider({ from: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000) }).catch(() => ({ items: [] })),
    ]);

    return (
      <div className="p-4 sm:p-6">
        <PageShell title={account.name} description={account.provider_kind}>
          <ProviderDetailClient
            account={account}
            credentials={credentials.items}
            health={health.items}
            models={models.items}
            usage={usageByProvider.items}
          />
        </PageShell>
      </div>
    );
  } catch (err: any) {
    if (err.status === 404) notFound();
    throw err;
  }
}