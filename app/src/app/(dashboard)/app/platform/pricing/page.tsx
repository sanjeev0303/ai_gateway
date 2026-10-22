import type { Metadata } from "next";

import {
  PageShell,
  StatCard,
} from "@/components/dashboard";
import { Coins, Layers, Calculator, ShieldCheck } from "lucide-react";
import { getApiClient } from "@/lib/session";
import { PricingClient, type PricingModelRow } from "./client";

export const metadata: Metadata = {
  title: "Pricing management · Sentinel",
  description: "Configure token rates, price sheet versions, and cost calculation rules per model.",
};

async function loadPricingData(): Promise<{
  rows: PricingModelRow[];
  totalModels: number;
  activeVersions: number;
  providerCount: number;
}> {
  try {
    const client = await getApiClient();
    const res = await client.pricing.list();
    const rows: PricingModelRow[] = (res.data ?? []).map((m) => ({
      id: m.id,
      model_slug: m.model_slug,
      provider_kind: m.provider_kind,
      display_name: m.display_name,
      currency: m.currency,
      status: m.status,
      created_at: m.created_at,
      updated_at: m.updated_at,
      latest_version: m.latest_version
        ? {
            id: m.latest_version.id,
            pricing_model_id: m.latest_version.pricing_model_id,
            version: m.latest_version.version,
            input_unit_price_micros: m.latest_version.input_unit_price_micros,
            output_unit_price_micros: m.latest_version.output_unit_price_micros,
            cache_read_input_token_price_micros:
              m.latest_version.cache_read_input_token_price_micros,
            effective_from: m.latest_version.effective_from,
            effective_to: m.latest_version.effective_to,
            created_at: m.latest_version.created_at,
          }
        : undefined,
    }));

    const providers = new Set(rows.map((r) => r.provider_kind));
    const activeVers = rows.filter((r) => !!r.latest_version).length;

    return {
      rows,
      totalModels: rows.length,
      activeVersions: activeVers,
      providerCount: providers.size,
    };
  } catch {
    return {
      rows: [],
      totalModels: 0,
      activeVersions: 0,
      providerCount: 0,
    };
  }
}

export default async function PricingPage() {
  const { rows, totalModels, activeVersions, providerCount } = await loadPricingData();

  return (
    <PageShell
      title="Pricing management"
      description="Define token rates, versioned price sheets, and cost calculation rules for your organization."
    >
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Pricing Models"
          value={String(totalModels)}
          hint="Registered model pricing sheets"
          icon={Coins}
        />
        <StatCard
          label="Active Versions"
          value={String(activeVersions)}
          hint="Models with effective price sheets"
          icon={Layers}
        />
        <StatCard
          label="Covered Providers"
          value={String(providerCount)}
          hint="Providers with pricing rules"
          icon={ShieldCheck}
        />
        <StatCard
          label="Calculation Mode"
          value="Integer Micros"
          hint="Zero floating-point rounding drift"
          icon={Calculator}
        />
      </div>

      <PricingClient initialModels={rows} />
    </PageShell>
  );
}
