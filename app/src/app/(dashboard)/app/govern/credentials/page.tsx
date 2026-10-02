import type { Metadata } from "next";

import { BarChart, DonutChart } from "@/components/charts";
import {
  DataTable,
  PageShell,
  StatCard,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatDate, formatNumber, formatRelative } from "@/lib/format";
import { NOW, credentials } from "@/lib/mock-data";
import {
  CREDENTIAL_STATUS_TONE,
  type CredentialKind,
  type StatusTone,
} from "@/types";

export const metadata: Metadata = {
  title: "Credentials · Sentinel",
  description: "Provider secrets, their rotation state, and which projects depend on each one.",
};

const KIND_TONE: Record<CredentialKind, StatusTone> = {
  api_key: "info",
  oauth: "success",
  service_account: "neutral",
  webhook: "warning",
};

interface CredentialRow {
  id: string;
  name: string;
  kind: string;
  kindTone: StatusTone;
  provider: string;
  masked: string;
  status: string;
  statusTone: StatusTone;
  created: string;
  expires: string;
  lastUsed: string;
  projects: string;
  projectCount: number;
}

const credentialColumns: readonly Column<CredentialRow>[] = [
  { key: "name", header: "Credential", primary: "name", secondary: "masked", sort: "name" },
  { key: "kind", header: "Kind", primary: "kind", tone: "kindTone", sort: "kind" },
  { key: "provider", header: "Provider", primary: "provider", sort: "provider" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  { key: "created", header: "Created", primary: "created", sort: "created" },
  { key: "expires", header: "Expires", primary: "expires", sort: "expires" },
  { key: "lastUsed", header: "Last used", primary: "lastUsed", sort: "lastUsed" },
  {
    key: "projects",
    header: "Projects",
    primary: "projects",
    align: "right",
    sort: "projectCount",
  },
];

const rows: CredentialRow[] = [...credentials]
  .sort((a, b) => a.name.localeCompare(b.name))
  .map((credential) => ({
    id: credential.id,
    name: credential.name,
    kind: credential.kind,
    kindTone: KIND_TONE[credential.kind],
    provider: credential.providerName,
    masked: credential.maskedValue,
    status: credential.status,
    statusTone: CREDENTIAL_STATUS_TONE[credential.status],
    created: formatDate(credential.createdAt),
    expires: credential.expiresAt ? formatDate(credential.expiresAt) : "Never",
    lastUsed: formatRelative(credential.lastUsedAt, NOW),
    projects: formatNumber(credential.usedByProjectCount),
    projectCount: credential.usedByProjectCount,
  }));

const healthy = credentials.filter((credential) => credential.status === "healthy");
const expiring = credentials.filter((credential) => credential.status === "expiring");
const broken = credentials.filter(
  (credential) => credential.status === "expired" || credential.status === "revoked",
);
const sharedWidely = credentials.filter(
  (credential) => credential.usedByProjectCount > 1,
);

const kindMix = (["api_key", "oauth", "service_account", "webhook"] as const).map(
  (kind) => ({ label: kind.replace("_", " "), value: credentials.filter((c) => c.kind === kind).length }),
);

const blastRadius = [...credentials]
  .sort((a, b) => b.usedByProjectCount - a.usedByProjectCount)
  .slice(0, 8)
  .map((credential) => ({
    label: credential.name,
    value: credential.usedByProjectCount,
  }));

export default function CredentialsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Credentials"
        description="Secrets are stored by the gateway and never returned to this console. Rotation is a scheduling problem, not a discovery problem."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Credentials"
            value={formatNumber(credentials.length)}
            hint={`${formatNumber(healthy.length)} healthy`}
          />
          <StatCard
            label="Expiring"
            value={formatNumber(expiring.length)}
            tone={expiring.length > 0 ? "warning" : "success"}
            hint="Rotation due"
          />
          <StatCard
            label="Unusable"
            value={formatNumber(broken.length)}
            tone={broken.length > 0 ? "danger" : "success"}
            hint="Expired or revoked"
          />
          <StatCard
            label="Shared"
            value={formatNumber(sharedWidely.length)}
            tone={sharedWidely.length > 0 ? "warning" : "neutral"}
            hint="Used by multiple projects"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Blast radius by credential</CardTitle>
              <CardDescription>
                A widely used secret is worth rotating on a schedule, because
                rotating it breaks everything at once.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={blastRadius}
                formatValue={(value) => `${formatNumber(value)} projects`}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Credentials by kind</CardTitle>
              <CardDescription>
                OAuth tokens outlive API keys and need an explicit revocation
                path.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={kindMix}
                centerLabel={formatNumber(credentials.length)}
                formatValue={(value) => `${formatNumber(value)} keys`}
              />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>All credentials</CardTitle>
            <CardDescription>
              Only the last four characters are ever shown, so a leak of this
              table is not a leak of your providers.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={credentialColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No credentials configured."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
