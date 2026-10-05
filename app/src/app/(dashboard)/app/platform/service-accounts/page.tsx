import type { Metadata } from "next";

import { DataTable, PageShell, StatCard, type Column } from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatDate, formatNumber } from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Service accounts · Sentinel",
  description: "Machine identities used by workloads instead of human sessions.",
};

const SERVICE_ACCOUNT_TONE: Record<string, StatusTone> = {
  active: "success",
  revoked: "neutral",
};

interface ServiceAccountRow {
  id: string;
  name: string;
  description: string;
  status: string;
  tone: StatusTone;
  created: string;
}

const COLUMNS: readonly Column<ServiceAccountRow>[] = [
  { key: "name", header: "Service account", primary: "name", secondary: "description", sort: "name" },
  { key: "status", header: "Status", primary: "status", tone: "tone", sort: "status" },
  { key: "created", header: "Created", primary: "created", sort: "created" },
];

async function loadServiceAccounts() {
  const { serviceAccounts } = await getApiClient();
  const list = await serviceAccounts.list();

  const rows: ServiceAccountRow[] = [...list.items]
    .sort((a, b) => a.name.localeCompare(b.name))
    .map((account) => ({
      id: account.id,
      name: account.name,
      description: account.description || "No description",
      status: account.status,
      tone: toneFor(SERVICE_ACCOUNT_TONE, account.status),
      created: formatDate(account.created_at),
    }));

  return {
    rows,
    total: rows.length,
    active: rows.filter((row) => row.status === "active").length,
    described: list.items.filter((item) => Boolean(item.description)).length,
  };
}

export default async function ServiceAccountsPage() {
  const { rows, total, active, described } = await loadServiceAccounts();

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Service accounts"
        description="Non-human identities that call the gateway on behalf of workloads and pipelines."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Service accounts"
            value={formatNumber(total)}
            hint={`${formatNumber(active)} active`}
          />
          <StatCard
            label="Revoked"
            value={formatNumber(total - active)}
            tone={total - active > 0 ? "neutral" : "success"}
            hint="No longer usable"
          />
          <StatCard
            label="Described"
            value={formatNumber(described)}
            hint="Have a description on file"
          />
          <StatCard label="Organization" value="1" hint="Accounts are org-wide" />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Accounts</CardTitle>
            <CardDescription>
              Service accounts hold no secret of their own. Keys are issued
              separately and can be scoped to a team or project.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No service accounts yet."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Usage and rotation</CardTitle>
            <CardDescription>Not reported by the control API today.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>
              A service account record carries no last-used timestamp, no key
              expiry, and no count of the projects using it, so this page cannot
              flag unused identities or schedule rotation.
            </p>
            <p>
              Expiry and owning team or project are recorded on the gateway
              credential itself, listed under Platform.
            </p>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
