import type { Metadata } from "next";
import { Plus } from "lucide-react";

import { DataTable, PageShell, StatCard, StatusBadge, type Column } from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatRelative, initials } from "@/lib/format";
import { credentials } from "@/lib/mock-data";
import { CREDENTIAL_STATUS_TONE, type StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Service accounts · Sentinel",
  description: "Machine identities used by workloads instead of human sessions.",
};

interface ServiceAccountRow {
  id: string;
  name: string;
  providerName: string;
  status: string;
  tone: StatusTone;
  lastUsed: string;
  projects: number;
  keyHint: string;
}

const COLUMNS: readonly Column<ServiceAccountRow>[] = [
  { key: "name", header: "Service account", primary: "name", secondary: "providerName" },
  { key: "status", header: "Status", primary: "status", tone: "tone" },
  { key: "keyHint", header: "Key", primary: "keyHint" },
  { key: "lastUsed", header: "Last used", primary: "lastUsed", align: "right" },
  { key: "projects", header: "Projects", primary: "projects", align: "right" },
];

export default function ServiceAccountsPage() {
  const serviceAccounts = credentials.filter(
    (credential) => credential.kind === "service_account" || credential.kind === "webhook",
  );

  const rows: ServiceAccountRow[] = serviceAccounts.map((credential) => ({
    id: credential.id,
    name: credential.name,
    providerName: credential.providerName,
    status: credential.status,
    tone: CREDENTIAL_STATUS_TONE[credential.status],
    lastUsed: formatRelative(credential.lastUsedAt),
    projects: credential.usedByProjectCount,
    keyHint: credential.maskedValue,
  }));

  const expiring = serviceAccounts.filter(
    (credential) => credential.status === "expiring" || credential.status === "expired",
  );
  const orphanCount = serviceAccounts.filter(
    (credential) => credential.usedByProjectCount === 0,
  ).length;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Service accounts"
        description="Non-human identities that call the gateway on behalf of workloads and pipelines."
        actions={
          <Button size="sm">
            <Plus aria-hidden /> New service account
          </Button>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard label="Service accounts" value={String(serviceAccounts.length)} hint="Machine identities" />
          <StatCard
            label="Healthy"
            value={String(serviceAccounts.length - expiring.length)}
            tone="success"
            hint="No action needed"
          />
          <StatCard
            label="Expiring"
            value={String(expiring.length)}
            tone={expiring.length > 0 ? "warning" : "success"}
            hint="Within the rotation window"
          />
          <StatCard
            label="Unreferenced"
            value={String(orphanCount)}
            tone={orphanCount > 0 ? "warning" : "neutral"}
            hint="Safe to revoke"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Accounts</CardTitle>
            <CardDescription>
              Secrets are shown masked at all times and never returned after creation.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              toolbar={
                <p className="text-sm text-muted-foreground">
                  {serviceAccounts.length} accounts across{" "}
                  {new Set(serviceAccounts.map((c) => c.providerName)).size} providers.
                </p>
              }
              emptyMessage="No service accounts yet."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Rotation policy</CardTitle>
            <CardDescription>
              Service account keys rotate automatically before they can expire.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="grid gap-3 sm:grid-cols-2">
              {serviceAccounts.slice(0, 4).map((credential) => (
                <li key={credential.id} className="flex items-center gap-3 rounded-lg border p-3">
                  <span className="grid size-9 shrink-0 place-items-center rounded-full bg-muted text-xs font-semibold">
                    {initials(credential.name)}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{credential.name}</p>
                    <p className="truncate text-xs text-muted-foreground">
                      {credential.providerName} ·{" "}
                      {credential.usedByProjectCount} project
                      {credential.usedByProjectCount === 1 ? "" : "s"}
                    </p>
                  </div>
                  <StatusBadge tone={CREDENTIAL_STATUS_TONE[credential.status]}>
                    {credential.status}
                  </StatusBadge>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
