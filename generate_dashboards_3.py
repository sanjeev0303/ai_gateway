import re
import os

files = {
    'apps/web/src/lib/bodies-billing.tsx': """import { MetricGrid, MetricCard, ChartCard, FilterBar, DateRangePicker } from "@/components/ui";

export function UsageAndSpendBody() {
  return (
    <div className="flex flex-col gap-6">
      <FilterBar>
        <DateRangePicker />
      </FilterBar>
      <MetricGrid>
        <MetricCard title="Current Spend" value="$0.00" />
        <MetricCard title="Forecast" value="$0.00" />
      </MetricGrid>
      <ChartCard title="Spend Trend">
        <div className="h-64 flex items-center justify-center text-muted-foreground border border-dashed rounded">Spend Chart Placeholder</div>
      </ChartCard>
    </div>
  );
}

export function SubscriptionBody() {
  return (
    <div className="flex flex-col gap-6">
      <div className="border border-line rounded-lg bg-surface p-8 text-center text-muted-foreground">
        Subscription Placeholder
      </div>
    </div>
  );
}

export function InvoicesBody() {
  return (
    <div className="flex flex-col gap-6">
      <div className="border border-line rounded-lg bg-surface p-8 text-center text-muted-foreground">
        Invoices Placeholder
      </div>
    </div>
  );
}
""",
    'apps/web/src/lib/bodies-developer.tsx': """import { CodeBlock } from "@/components/ui";

export function QuickStartBody() {
  return (
    <div className="flex flex-col gap-6">
      <h2 className="text-xl font-semibold">Base URL</h2>
      <CodeBlock code="https://gateway.example.com/v1" />
      
      <h2 className="text-xl font-semibold mt-4">Authentication</h2>
      <p className="text-muted-foreground text-sm">Pass your gateway credential in the Authorization header.</p>
      <CodeBlock code='Authorization: Bearer <your-credential>' />
      
      <h2 className="text-xl font-semibold mt-4">Example Request (cURL)</h2>
      <CodeBlock code={`curl https://gateway.example.com/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <your-credential>" \\
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'`} language="bash" />
    </div>
  );
}

export function SdksBody() {
  return (
    <div className="flex flex-col gap-6">
      <div className="border border-line rounded-lg bg-surface p-8 text-center text-muted-foreground">SDKs Documentation Placeholder</div>
    </div>
  );
}

export function CliBody() {
  return (
    <div className="flex flex-col gap-6">
      <div className="border border-line rounded-lg bg-surface p-8 text-center text-muted-foreground">CLI Documentation Placeholder</div>
    </div>
  );
}

export function ApiDocsBody() {
  return (
    <div className="flex flex-col gap-6">
      <div className="border border-line rounded-lg bg-surface p-8 text-center text-muted-foreground">API Reference Placeholder</div>
    </div>
  );
}
""",
    'apps/web/src/lib/sections.tsx': """import { SpecView } from "@/lib/spec";
import { MetricGrid, MetricCard, DataTable, Avatar, UserPresence, StatusBadge, ProviderBadge } from "@/components/ui";
import { useEffect, useState } from "react";
import { listApiCredentials, listGatewayCredentials, listInvitations, listMembers, listProjects, listProviderAccounts, listProviderHealth, listServiceAccounts, listTeams } from "@/lib/control-api";

export function ApiCredentialsBody() {
  const [items, setItems] = useState<any[]>([]);
  useEffect(() => { listApiCredentials().then(r => setItems(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[{ key: "name", header: "Name", render: (r: any) => r.name }]}
        rows={items} isLoading={false} isError={false} error={null} empty="No API credentials." caption="API Credentials"
      />
    </div>
  );
}

export function GatewayCredentialsBody() {
  const [items, setItems] = useState<any[]>([]);
  useEffect(() => { listGatewayCredentials().then(r => setItems(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[{ key: "name", header: "Name", render: (r: any) => r.name }]}
        rows={items} isLoading={false} isError={false} error={null} empty="No Gateway credentials." caption="Gateway Credentials"
      />
    </div>
  );
}

export function InvitationsBody() {
  const [items, setItems] = useState<any[]>([]);
  useEffect(() => { listInvitations().then(r => setItems(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[
          { key: "email", header: "Email", render: (r: any) => r.email },
          { key: "role", header: "Role", render: (r: any) => r.role },
          { key: "status", header: "Status", render: (r: any) => <StatusBadge status={r.status === 'pending' ? 'warning' : 'neutral'} label={r.status} /> }
        ]}
        rows={items} isLoading={false} isError={false} error={null} empty="No pending invitations." caption="Invitations"
      />
    </div>
  );
}

export function MembersBody() {
  const [items, setItems] = useState<any[]>([]);
  useEffect(() => { listMembers().then(r => setItems(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[
          { key: "user", header: "User", render: (r: any) => <UserPresence name={r.user?.display_name || r.user?.email || "Unknown"} online={false} /> },
          { key: "role", header: "Role", render: (r: any) => r.role }
        ]}
        rows={items} isLoading={items.length === 0} isError={false} error={null} empty="No members found." caption="Members"
      />
    </div>
  );
}

export function ModelsBody() {
  return <div className="p-8 text-center text-muted-foreground border border-line rounded-lg">Model Registry Placeholder</div>;
}

export function ProjectsBody() {
  const [items, setItems] = useState<any[]>([]);
  useEffect(() => { listProjects().then(r => setItems(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[
          { key: "name", header: "Project", render: (r: any) => r.name },
          { key: "status", header: "Status", render: (r: any) => r.status }
        ]}
        rows={items} isLoading={items.length === 0} isError={false} error={null} empty="No projects found." caption="Projects"
      />
    </div>
  );
}

export function ProviderAccountsBody() {
  const [items, setItems] = useState<any[]>([]);
  useEffect(() => { listProviderAccounts().then(r => setItems(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[
          { key: "name", header: "Provider Account", render: (r: any) => <ProviderBadge provider={r.name || r.provider_kind} /> },
          { key: "status", header: "Status", render: (r: any) => r.status }
        ]}
        rows={items} isLoading={false} isError={false} error={null} empty="No provider accounts." caption="Provider Accounts"
      />
    </div>
  );
}

export function ProviderHealthBody() {
  const [items, setItems] = useState<any[]>([]);
  useEffect(() => { listProviderHealth().then(r => setItems(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[
          { key: "provider", header: "Provider", render: (r: any) => <ProviderBadge provider={r.provider_kind} /> },
          { key: "status", header: "Status", render: (r: any) => r.status }
        ]}
        rows={items} isLoading={false} isError={false} error={null} empty="No provider health data." caption="Provider Health"
      />
    </div>
  );
}

export function ServiceAccountsBody() {
  const [items, setItems] = useState<any[]>([]);
  useEffect(() => { listServiceAccounts().then(r => setItems(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[
          { key: "name", header: "Service Account", render: (r: any) => r.name },
          { key: "status", header: "Status", render: (r: any) => r.status }
        ]}
        rows={items} isLoading={false} isError={false} error={null} empty="No service accounts." caption="Service Accounts"
      />
    </div>
  );
}

export function TeamsBody() {
  const [items, setItems] = useState<any[]>([]);
  useEffect(() => { listTeams().then(r => setItems(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[
          { key: "name", header: "Team Name", render: (r: any) => r.name }
        ]}
        rows={items} isLoading={items.length === 0} isError={false} error={null} empty="No teams found." caption="Teams"
      />
    </div>
  );
}
"""
}

for path, content in files.items():
    with open(path, 'w') as f:
        f.write(content)

print("Updated billing, developer, and sections bodies.")
