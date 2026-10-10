import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { PageShell, DataTable, type Column } from "@/components/dashboard";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { getApiClient } from "@/lib/session";
import { formatDate, formatNumber, microsToUsd, formatUsd } from "@/lib/format";
import { toneFor } from "@/lib/org-tones";

export const metadata: Metadata = {
  title: "Service account details · Sentinel",
};

interface ServiceAccountPageProps {
  params: { id: string };
}

const CREDENTIAL_COLUMNS: readonly Column<any>[] = [
  { key: "name", header: "Credential", primary: "name", sort: "name" },
  { key: "status", header: "Status", primary: "status", tone: "tone", sort: "status" },
  { key: "created", header: "Created", primary: "created", sort: "created" },
];

export default async function ServiceAccountDetailPage({ params }: ServiceAccountPageProps) {
  const { serviceAccounts, credentials, usage } = await getApiClient();

  // Load account
  const list = await serviceAccounts.list();
  const account = list.items.find((a) => a.id === params.id);
  if (!account) notFound();

  // Load credentials associated with this account
  const credsList = await credentials.listGateway();
  const associatedCreds = credsList.items
    .filter((c) => c.subject_id === account.id && c.subject_type === "service_account")
    .map((c) => ({
      id: c.id,
      name: c.name,
      status: c.status,
      tone: c.status === "active" ? "success" : "neutral",
      created: formatDate(c.created_at),
    }));

  // Load usage for this account
  const usageRes = await usage.byServiceAccount();
  const attributed = usageRes.items.find((row) => row.ServiceAccountID === account.id);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title={account.name}
        description={account.description || "No description provided."}
      >
        <Card className="mb-4">
          <CardHeader>
            <CardTitle>Usage Summary (30d)</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
              <div>
                <dt className="text-muted-foreground">Requests</dt>
                <dd className="text-2xl font-medium">{formatNumber(attributed?.Requests ?? 0)}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Tokens</dt>
                <dd className="text-2xl font-medium">{formatNumber(attributed?.TotalTokens ?? 0)}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Spend</dt>
                <dd className="text-2xl font-medium">{formatUsd(microsToUsd(attributed?.CostMicros ?? 0))}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Status</dt>
                <dd className="text-2xl font-medium capitalize">{account.status}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Associated Credentials</CardTitle>
            <CardDescription>Gateway credentials authenticated as this machine identity.</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={CREDENTIAL_COLUMNS}
              rows={associatedCreds}
              rowKey="id"
              pageSize={10}
              emptyMessage="No credentials exist for this service account."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
