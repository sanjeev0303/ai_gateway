import type { Metadata } from "next";

import type { Model } from "@ai-governance/api-client";
import { DataTable, type Column } from "@ai-governance/console-ui/dashboard/data-table";
import { PageShell } from "@ai-governance/console-ui/dashboard/page-shell";
import { ErrorState } from "@ai-governance/console-ui/dashboard/state";
import { formatCompact, formatNumber, formatProviderName } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@ai-governance/console-ui/types/common";
import { ModelActionsCell, type ModelActionRow } from "./model-actions";

export const metadata: Metadata = { title: "Model catalog · Platform console" };

interface ModelRow extends ModelActionRow {
  slug: string;
  provider: string;
  status: string;
  statusTone: StatusTone;
  context: string;
  features: string;
}

const COLUMNS: Column<ModelRow>[] = [
  { key: "name", header: "Model", primary: "name", secondary: "slug", grow: true },
  { key: "provider", header: "Provider", primary: "provider" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "context", header: "Context", primary: "context", align: "right" },
  { key: "features", header: "Supports", primary: "features" },
];

function features(model: Model): string {
  const list: string[] = [];
  if (model.supports_streaming) list.push("streaming");
  if (model.supports_tools) list.push("tools");
  if (model.supports_reasoning) list.push("reasoning");
  return list.length > 0 ? list.join(", ") : "—";
}

export default async function AdminModelsPage() {
  let models: Model[];
  try {
    const { admin } = await getApiClient();
    models = (await admin.models()).items;
  } catch {
    return (
      <PageShell title="Model catalog">
        <ErrorState description="The model catalog could not be loaded. Check that the control API is running, then reload." />
      </PageShell>
    );
  }

  const rows: ModelRow[] = models.map((m) => ({
    id: m.id,
    name: m.display_name || m.model_slug,
    slug: m.model_slug,
    active: m.is_active && m.status !== "inactive",
    provider: formatProviderName(m.provider_kind),
    status: m.is_active && m.status !== "inactive" ? "Active" : "Inactive",
    statusTone: m.is_active && m.status !== "inactive" ? "success" : "neutral",
    context: m.context_limit ? formatCompact(m.context_limit) : "—",
    features: features(m),
  }));
  const active = rows.filter((r) => r.active).length;

  return (
    <PageShell
      title="Model catalog"
      description={`${formatNumber(active)} of ${formatNumber(rows.length)} models active. The catalog is shared by every organization; an organization’s sync can add models but never switch one off.`}
    >
      <DataTable
        columns={COLUMNS}
        rows={rows}
        rowKey="id"
        pageSize={25}
        ActionsComponent={ModelActionsCell}
        emptyMessage="The catalog is empty. Models are added when an organization syncs a provider account."
      />
    </PageShell>
  );
}
