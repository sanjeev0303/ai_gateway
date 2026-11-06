import type { Metadata } from "next";

import {
  DataTable,
  ObserveFilters,
  PageShell,
  StatCard,
  type Column,
  BreakdownTabs,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  formatNumber,
  formatPercent,
  microsToUsd,
} from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Providers · Tollseal",
  description: "Upstream connections, and the cost of every provider.",
};

const ACCOUNT_STATUS_TONE: Record<string, StatusTone> = {
  active: "success",
  disabled: "neutral",
};

interface ProviderRow {
  id: string;
  name: string;
  kind: string;
  status: string;
  statusTone: StatusTone;
  requests: number;
  tokens: number;
  share: string;
  shareFraction: number;
  spend: number;
  detailUrl: string;
}

const providerColumns: readonly Column<ProviderRow>[] = [
  { key: "name", header: "Connection", primary: "name", secondary: "kind", sort: "name", href: "detailUrl" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  {
    key: "requests",
    header: "Requests",
    primary: "requests",
    format: "compact",
    align: "right",
    sort: "requests",
  },
  {
    key: "tokens",
    header: "Tokens",
    primary: "tokens",
    format: "compact",
    align: "right",
    sort: "tokens",
  },
  {
    key: "share",
    header: "Share",
    primary: "share",
    meter: "shareFraction",
    meterMax: 1,
    sort: "shareFraction",
    align: "right",
  },
  {
    key: "spend",
    header: "Spend 30d",
    primary: "spend",
    format: "usd",
    align: "right",
    sort: "spend",
  },
];

const DAY = 24 * 60 * 60 * 1000;

function trailingWindow() {
  return { from: new Date(Date.now() - 30 * DAY) };
}

export default async function ProvidersPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const filters = {
    project_id: typeof params.project_id === "string" ? params.project_id : undefined,
    user_id: typeof params.user_id === "string" ? params.user_id : undefined,
    team_id: typeof params.team_id === "string" ? params.team_id : undefined,
    service_account_id: typeof params.service_account_id === "string" ? params.service_account_id : undefined,
    provider: typeof params.provider === "string" ? params.provider : undefined,
    model: typeof params.model === "string" ? params.model : undefined,
    status: typeof params.status === "string" ? params.status : undefined,
  };

  const { providers, usage } = await getApiClient();
  const window = { ...trailingWindow(), ...filters };
  const [accounts, providerUsage] = await Promise.all([
    providers.listAccounts(),
    usage.byProvider(window),
  ]);

  // by-provider groups on provider_kind, which is a real key on both sides.
  const usageByKind = new Map(providerUsage.items.map((row) => [row.Provider, row]));

  const totalSpend = providerUsage.items.reduce(
    (sum, row) => sum + microsToUsd(row.CostMicros),
    0,
  );
  const active = accounts.items.filter((account) => account.status === "active").length;

  const rows: ProviderRow[] = accounts.items.map((account) => {
    const attributed = usageByKind.get(account.provider_kind);
    const spend = microsToUsd(attributed?.CostMicros ?? 0);
    const shareFraction = totalSpend > 0 ? spend / totalSpend : 0;
    return {
      id: account.id,
      name: account.name,
      kind: account.provider_kind,
      status: account.status,
      statusTone: toneFor(ACCOUNT_STATUS_TONE, account.status),
      requests: attributed?.Requests ?? 0,
      tokens: attributed?.TotalTokens ?? 0,
      share: formatPercent(shareFraction),
      shareFraction,
      spend,
      detailUrl: `/app/platform/provider-accounts/${account.id}`,
    };
  });

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Providers"
        description="Upstream connections, and what each one costs."
      >
        <BreakdownTabs />
        <ObserveFilters />
        <section className="grid gap-4 sm:grid-cols-2">
          <StatCard
            label="Connections"
            value={formatNumber(accounts.items.length)}
            hint={`${formatNumber(active)} active`}
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Connections</CardTitle>
            <CardDescription>
              Every upstream the gateway holds credentials for.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={providerColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No providers connected."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
