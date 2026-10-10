import { notFound } from "next/navigation";
import { getApiClient } from "@/lib/session";
import { PageShell } from "@/components/dashboard";
import { ProviderDetailClient } from "./provider-detail-client";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Provider Account Detail · Sentinel",
};

export default async function ProviderAccountDetailPage({ params }: { params: { id: string } }) {
  const api = await getApiClient();

  try {
    const account = await api.providers.getAccount(params.id);
    const credentials = await api.providers.listCredentials();
    const health = await api.providers.health();

    return (
      <div className="p-4 sm:p-6">
        <PageShell title={account.name} description={account.provider_kind}>
          <ProviderDetailClient account={account} credentials={credentials.items} health={health.items} />
        </PageShell>
      </div>
    );
  } catch (err: any) {
    if (err.status === 404) notFound();
    throw err;
  }
}