"use client";

import { useMemo } from "react";

import { DataTable, Mono, StatusPill, never, type Column } from "@/components/ui";
import {
  useProviderAccounts,
  useProviderCredentials,
  useProviderHealth,
} from "@/lib/queries";
import type { ProviderAccount } from "@/lib/control-api";
import { PendingColumns } from "@/lib/spec";

interface ProviderRow {
  readonly id: string;
  readonly provider: string;
  readonly name: string;
  readonly connection: string;
  readonly status: string;
  readonly health: string;
  readonly checkedAt: string;
  readonly fingerprint: string;
}

export function ProvidersObserveBody() {
  const accounts = useProviderAccounts();
  const health = useProviderHealth();
  const credentials = useProviderCredentials();

  const rows = useMemo<readonly ProviderRow[]>(
    () =>
      (accounts.data ?? []).map((account: ProviderAccount) => {
        const state = health.data?.find((h) => h.provider_kind === account.provider_kind);
        const own = (credentials.data ?? []).filter(
          (c) => c.provider_account_id === account.id,
        );
        return {
          id: account.id,
          provider: account.provider_kind,
          name: account.name,
          connection: account.base_url ?? "default endpoint",
          status: account.status,
          health: state?.status ?? "unknown",
          checkedAt: state?.checked_at ?? "",
          fingerprint: own.length === 0 ? "none" : own.map((c) => c.fingerprint).join(", "),
        };
      }),
    [accounts.data, health.data, credentials.data],
  );

  const columns: readonly Column<ProviderRow>[] = [
    { key: "provider", header: "Provider", render: (r) => <StatusPill value={r.provider} /> },
    { key: "name", header: "Account", render: (r) => r.name },
    { key: "connection", header: "Connection", render: (r) => r.connection },
    { key: "status", header: "Status", render: (r) => <StatusPill value={r.status} /> },
    { key: "health", header: "Health", render: (r) => <StatusPill value={r.health} /> },
    { key: "checked", header: "Last health check", render: (r) => never(r.checkedAt, "Not checked") },
    { key: "fingerprint", header: "Credential fingerprint", render: (r) => <Mono>{r.fingerprint}</Mono> },
  ];

  return (
    <>
      <DataTable
        caption="Provider accounts, health, and credential fingerprints"
        columns={columns}
        rows={rows}
        isLoading={accounts.isPending || health.isPending || credentials.isPending}
        isError={accounts.isError || health.isError || credentials.isError}
        error={accounts.error ?? health.error ?? credentials.error}
        empty="No provider accounts yet."
      />
      <PendingColumns
        subject="Per-provider"
        pending={["Models", "Requests", "Tokens", "Spend", "Errors"]}
        needs={["GET /v1/usage/aggregates?provider", "GET /v1/requests/aggregates"]}
      />
      <p className="dash-note">
        Only credential fingerprints are shown. The stored secret cannot be read back through the
        control API at any point.
      </p>
    </>
  );
}
