import re
import os

files = {
    'apps/web/src/lib/bodies-providers.tsx': """import { MetricGrid, MetricCard, DataTable, ProviderBadge, HealthIndicator, StatusBadge } from "@/components/ui";
import { useEffect, useState } from "react";
import { listProviderAccounts, listProviderHealth } from "@/lib/control-api";

export function ProvidersObserveBody() {
  const [providers, setProviders] = useState<any[]>([]);
  const [health, setHealth] = useState<any[]>([]);
  
  useEffect(() => {
    listProviderAccounts().then(r => setProviders(r.items || []));
    listProviderHealth().then(r => setHealth(r.items || []));
  }, []);
  
  return (
    <div className="flex flex-col gap-6">
      <MetricGrid>
        <MetricCard title="Total Providers" value={providers.length} />
        <MetricCard title="Healthy Providers" value={health.filter((h: any) => h.status === 'healthy').length} />
      </MetricGrid>
      
      <div className="border border-line rounded-lg overflow-hidden bg-surface">
        <DataTable
          columns={[
            { key: "name", header: "Provider Account", render: (r: any) => <ProviderBadge provider={r.name || r.provider_kind} /> },
            { key: "status", header: "Status", render: (r: any) => <StatusBadge status={r.status === 'active' ? 'success' : 'neutral'} label={r.status} /> },
            { key: "created", header: "Added", render: (r: any) => new Date(r.created_at).toLocaleDateString() }
          ]}
          rows={providers}
          isLoading={false}
          isError={false}
          error={null}
          empty="No providers configured."
          caption="Providers"
        />
      </div>
    </div>
  );
}
""",
    'apps/web/src/lib/bodies-govern.tsx': """import { MetricGrid, MetricCard, DataTable, StatusBadge, ProviderBadge, ModelBadge, FilterBar, SearchInput } from "@/components/ui";
import { useEffect, useState } from "react";
import { listGatewayCredentials } from "@/lib/control-api";

export function PoliciesBody() {
  return (
    <div className="flex flex-col gap-6">
      <MetricGrid>
        <MetricCard title="Active Policies" value="4" />
        <MetricCard title="Violations (24h)" value="0" />
      </MetricGrid>
      <div className="border border-line rounded-lg overflow-hidden bg-surface p-8 text-center text-muted-foreground">
        Visual Policy Builder Placeholder
      </div>
    </div>
  );
}

export function ModelAccessBody() {
  return (
    <div className="flex flex-col gap-6">
      <div className="border border-line rounded-lg overflow-hidden bg-surface">
        <DataTable
          columns={[
            { key: "model", header: "Model", render: (r: any) => <ModelBadge model={r.model} /> },
            { key: "scope", header: "Scope", render: (r: any) => r.scope },
            { key: "access", header: "Access", render: (r: any) => <StatusBadge status={r.access ? "success" : "error"} label={r.access ? "Allowed" : "Denied"} /> }
          ]}
          rows={[
            { model: "gpt-4", scope: "Organization", access: true },
            { model: "claude-3-opus", scope: "Team: Engineering", access: true }
          ]}
          isLoading={false}
          isError={false}
          error={null}
          empty="No access rules found."
          caption="Model Access"
        />
      </div>
    </div>
  );
}

export function BudgetsLimitsBody() {
  return (
    <div className="flex flex-col gap-6">
       <MetricGrid>
        <MetricCard title="Active Budgets" value="2" />
      </MetricGrid>
      <div className="border border-line rounded-lg overflow-hidden bg-surface p-8 text-center text-muted-foreground">
        Budgets & Limits Placeholder
      </div>
    </div>
  );
}

export function GuardrailsBody() {
  return (
    <div className="flex flex-col gap-6">
      <MetricGrid>
        <MetricCard title="Active Guardrails" value="0" />
      </MetricGrid>
      <div className="border border-line rounded-lg overflow-hidden bg-surface p-8 text-center text-muted-foreground">
        Guardrails Placeholder
      </div>
    </div>
  );
}

export function CredentialsBody() {
  const [credentials, setCredentials] = useState<any[]>([]);
  useEffect(() => { listGatewayCredentials().then(r => setCredentials(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <FilterBar>
        <SearchInput placeholder="Search credentials..." />
      </FilterBar>
      <div className="border border-line rounded-lg overflow-hidden bg-surface">
        <DataTable
          columns={[
            { key: "name", header: "Name", render: (r: any) => r.name },
            { key: "prefix", header: "Prefix", render: (r: any) => <code className="text-xs">{r.key_prefix}...</code> },
            { key: "subject", header: "Subject", render: (r: any) => r.subject_type },
            { key: "status", header: "Status", render: (r: any) => <StatusBadge status={r.status === 'active' ? 'success' : 'error'} label={r.status} /> }
          ]}
          rows={credentials}
          isLoading={false}
          isError={false}
          error={null}
          empty="No credentials found."
          caption="Credentials"
        />
      </div>
    </div>
  );
}
""",
    'apps/web/src/lib/bodies-security.tsx': """import { MetricGrid, MetricCard, DataTable, StatusBadge, FilterBar, DateRangePicker } from "@/components/ui";

export function AuditLogsBody() {
  return (
    <div className="flex flex-col gap-6">
      <FilterBar>
        <DateRangePicker />
      </FilterBar>
      <div className="border border-line rounded-lg overflow-hidden bg-surface p-8 text-center text-muted-foreground">
        Audit Logs Placeholder
      </div>
    </div>
  );
}

export function AlertsBody() {
  return (
    <div className="flex flex-col gap-6">
      <MetricGrid>
        <MetricCard title="Active Alerts" value="0" />
      </MetricGrid>
      <div className="border border-line rounded-lg overflow-hidden bg-surface p-8 text-center text-muted-foreground">
        Alerts Placeholder
      </div>
    </div>
  );
}

export function SecurityEventsBody() {
  return (
    <div className="flex flex-col gap-6">
      <MetricGrid>
        <MetricCard title="Security Events (24h)" value="0" />
      </MetricGrid>
      <div className="border border-line rounded-lg overflow-hidden bg-surface p-8 text-center text-muted-foreground">
        Security Events Placeholder
      </div>
    </div>
  );
}
"""
}

for path, content in files.items():
    with open(path, 'w') as f:
        f.write(content)

print("Updated providers, govern, and security bodies.")
