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
import { formatDate, formatRelative } from "@/lib/format";
import { credentials } from "@/lib/mock-data";
import { CREDENTIAL_STATUS_TONE, type StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Gateway credentials · Sentinel",
  description: "Every credential stored by the gateway, with expiry and rotation state.",
};

interface CredentialRow {
  id: string;
  name: string;
  kind: string;
  providerName: string;
  status: string;
  tone: StatusTone;
  masked: string;
  created: string;
  expires: string;
  lastUsed: string;
  usedBy: number;
}

const COLUMNS: readonly Column<CredentialRow>[] = [
  { key: "name", header: "Credential", primary: "name", secondary: "kind" },
  { key: "status", header: "Status", primary: "status", tone: "tone" },
  { key: "providerName", header: "Bound to", primary: "providerName" },
  { key: "masked", header: "Secret", primary: "masked" },
  { key: "expires", header: "Expires", primary: "expires", align: "right" },
  { key: "lastUsed", header: "Last used", primary: "lastUsed", align: "right" },
  { key: "usedBy", header: "Projects", primary: "usedBy", align: "right" },
];

export default function GatewayCredentialsPage() {
  const rows: CredentialRow[] = credentials.map((credential) => ({
    id: credential.id,
    name: credential.name,
    kind: credential.kind.replace(/_/g, " "),
    providerName: credential.providerName,
    status: credential.status,
    tone: CREDENTIAL_STATUS_TONE[credential.status],
    masked: credential.maskedValue,
    created: formatDate(credential.createdAt),
    expires: credential.expiresAt ? formatDate(credential.expiresAt) : "Never",
    lastUsed: formatRelative(credential.lastUsedAt),
    usedBy: credential.usedByProjectCount,
  }));

  const expiring = credentials.filter((c) => c.status === "expiring");
  const expired = credentials.filter((c) => c.status === "expired");
  const soonest = [...credentials]
    .filter((c) => c.expiresAt)
    .sort((a, b) => (a.expiresAt ?? "").localeCompare(b.expiresAt ?? ""))[0];

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Gateway credentials"
        description="All secrets the gateway holds, whether for upstream providers or inbound webhooks."
        actions={
          <Button size="sm">
            <Plus aria-hidden /> Add credential
          </Button>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard label="Total credentials" value={String(credentials.length)} hint="Across all providers" />
          <StatCard
            label="Healthy"
            value={String(credentials.length - expiring.length - expired.length)}
            tone="success"
            hint="No action required"
          />
          <StatCard
            label="Expiring soon"
            value={String(expiring.length)}
            tone={expiring.length > 0 ? "warning" : "success"}
            hint={soonest ? `Next: ${formatDate(soonest.expiresAt!)}` : "Nothing scheduled"}
          />
          <StatCard
            label="Expired"
            value={String(expired.length)}
            tone={expired.length > 0 ? "danger" : "neutral"}
            hint="Rotation has lapsed"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Stored credentials</CardTitle>
            <CardDescription>
              Secrets are masked on read and can only be revealed once, at creation.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              toolbar={
                <p className="text-sm text-muted-foreground">
                  {credentials.filter((c) => c.usedByProjectCount > 0).length} of{" "}
                  {credentials.length} are referenced by at least one project.
                </p>
              }
              emptyMessage="No credentials stored yet."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Rotation posture</CardTitle>
            <CardDescription>
              Expired or expiring secrets, ordered by urgency.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {expiring.length + expired.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                Every credential is inside its rotation window.
              </p>
            ) : (
              <ul className="space-y-2">
                {[...expired, ...expiring].map((credential) => (
                  <li
                    key={credential.id}
                    className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3"
                  >
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">{credential.name}</p>
                      <p className="truncate text-xs text-muted-foreground">
                        {credential.providerName} ·{" "}
                        {credential.expiresAt
                          ? `expires ${formatDate(credential.expiresAt)}`
                          : "no expiry set"}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      <StatusBadge tone={CREDENTIAL_STATUS_TONE[credential.status]}>
                        {credential.status}
                      </StatusBadge>
                      <Button size="sm" variant="outline">
                        Rotate
                      </Button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
