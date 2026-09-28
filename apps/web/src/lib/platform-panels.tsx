"use client";

import Link from "next/link";

import { DataTable, StatusPill, formatDateTime, type Column } from "@/components/ui";
import {
  useMembers,
  useModels,
  useProjects,
  useProviderAccounts,
  useProviderCredentials,
  useProviderHealth,
  useServiceAccounts,
} from "@/lib/queries";
import type { ProviderHealth } from "@/lib/control-api";

export function ProviderHealthPanel() {
  const query = useProviderHealth();
  const columns: readonly Column<ProviderHealth>[] = [
    {
      key: "provider",
      header: "Provider",
      render: (h) => <StatusPill value={h.provider_kind} />,
    },
    { key: "status", header: "Status", render: (h) => <StatusPill value={h.status} /> },
    {
      key: "detail",
      header: "Detail",
      render: (h) => h.status_reason || h.last_error || "—",
    },
    { key: "checked", header: "Last checked", render: (h) => formatDateTime(h.checked_at) },
  ];

  return (
    <section className="dash-panel">
      <h2>Provider health</h2>
      <DataTable
        caption="Provider health"
        columns={columns}
        rows={query.data ?? []}
        isLoading={query.isPending}
        isError={query.isError}
        error={query.error}
        empty="No provider health checks recorded yet."
      />
    </section>
  );
}

export function SetupPanel() {
  const accounts = useProviderAccounts();
  const credentials = useProviderCredentials();
  const models = useModels();
  const serviceAccounts = useServiceAccounts();
  const members = useMembers();
  const projects = useProjects();

  const steps = [
    { label: "Add a member", done: (members.data?.length ?? 0) > 1, href: "/dashboard/members" },
    { label: "Create a project", done: (projects.data?.length ?? 0) > 0, href: "/dashboard/projects" },
    {
      label: "Register a provider account",
      done: (accounts.data?.length ?? 0) > 0,
      href: "/dashboard/provider-accounts",
    },
    {
      label: "Store a provider secret",
      done: (credentials.data?.length ?? 0) > 0,
      href: "/dashboard/provider-credentials",
    },
    { label: "Sync the model catalog", done: (models.data?.length ?? 0) > 0, href: "/dashboard/models" },
    {
      label: "Create a service account for automation",
      done: (serviceAccounts.data?.length ?? 0) > 0,
      href: "/dashboard/service-accounts",
    },
  ];

  return (
    <section className="dash-panel">
      <h2>Setup</h2>
      <ul className="checklist">
        {steps.map((step) => (
          <li key={step.label}>
            <span data-done={step.done || undefined} className="check-mark" aria-hidden>
              {step.done ? "✓" : "○"}
            </span>
            {step.done ? <span className="check-label">{step.label}</span> : <Link href={step.href}>{step.label}</Link>}
          </li>
        ))}
      </ul>
    </section>
  );
}
