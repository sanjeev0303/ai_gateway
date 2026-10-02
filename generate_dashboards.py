import re
import os

files = {
    'apps/web/src/lib/bodies-observe.tsx': """import { useEffect, useState } from "react";
import { ActivityEvent, listActivity, subscribeToActivity, getUsageSummary, getUsageTimeSeries } from "@/lib/control-api";
import { MetricGrid, MetricCard, TrendIndicator, ChartCard, DataTable, StatusBadge, ProviderBadge, ModelBadge, TokenDisplay, CostDisplay, FilterBar, DateRangePicker, Select, SearchInput } from "@/components/ui";

export function LiveActivityBody() {
  const [events, setEvents] = useState<ActivityEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    let mounted = true;
    listActivity().then((res) => { if (mounted) { setEvents(res.data || []); setLoading(false); } })
      .catch((err) => { if (mounted) { setError(err); setLoading(false); } });
    const unsubscribe = subscribeToActivity((event) => {
        if (!mounted) return;
        setEvents((prev) => {
          if (event.current_state === "completed" || event.current_state === "failed") {
            return prev.filter((e) => e.request_id !== event.request_id);
          }
          const idx = prev.findIndex((e) => e.request_id === event.request_id);
          if (idx >= 0) { const next = [...prev]; next[idx] = event; return next; }
          return [event, ...prev];
        });
      }, (err) => console.error(err));
    return () => { mounted = false; unsubscribe(); };
  }, []);

  if (loading) return <div className="p-8 text-center">Loading activity...</div>;
  if (error) return <div className="p-8 text-rose-500">{error.message}</div>;

  return (
    <div className="flex flex-col gap-6">
      <MetricGrid>
        <MetricCard title="Active Requests" value={events.length} />
        <MetricCard title="Active Users" value={new Set(events.map(e => e.user_id).filter(Boolean)).size} />
        <MetricCard title="Current Models" value={new Set(events.map(e => e.model).filter(Boolean)).size} />
        <MetricCard title="Current Projects" value={new Set(events.map(e => e.project_id).filter(Boolean)).size} />
      </MetricGrid>
      <div className="border border-line rounded-lg overflow-hidden bg-surface">
        <DataTable
          columns={[
            { key: "id", header: "Request ID", render: (e) => <span className="font-mono text-xs">{e.request_id.split("-")[0]}</span> },
            { key: "user", header: "User", render: (e) => e.user_name || e.user_id || "Unknown" },
            { key: "provider", header: "Provider", render: (e) => <ProviderBadge provider={e.provider} /> },
            { key: "model", header: "Model", render: (e) => <ModelBadge model={e.model} /> },
            { key: "started", header: "Started", render: (e) => new Date(e.started_at).toLocaleTimeString() }
          ]}
          rows={events}
          isLoading={false}
          isError={false}
          error={null}
          empty="No active requests."
          caption="Live Activity"
        />
      </div>
    </div>
  );
}

export function UsageBody() {
  const [data, setData] = useState<any>(null);
  
  useEffect(() => {
    getUsageSummary().then(setData).catch(console.error);
  }, []);

  return (
    <div className="flex flex-col gap-6">
      <FilterBar>
        <DateRangePicker />
        <Select options={[{label: "All Providers", value: ""}]} />
        <Select options={[{label: "All Models", value: ""}]} />
      </FilterBar>
      <MetricGrid>
        <MetricCard title="Requests" value={data ? data.Requests : "..."} />
        <MetricCard title="Input Tokens" value={data ? data.InputTokens : "..."} />
        <MetricCard title="Output Tokens" value={data ? data.OutputTokens : "..."} />
        <MetricCard title="Total Tokens" value={data ? data.TotalTokens : "..."} />
      </MetricGrid>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <ChartCard title="Token Trend">
          <div className="h-64 flex items-center justify-center text-muted-foreground border border-dashed rounded">Chart Data Placeholder</div>
        </ChartCard>
        <ChartCard title="Request Trend">
          <div className="h-64 flex items-center justify-center text-muted-foreground border border-dashed rounded">Chart Data Placeholder</div>
        </ChartCard>
      </div>
    </div>
  );
}

export function CostsBody() {
  const [data, setData] = useState<any>(null);
  
  useEffect(() => {
    getUsageSummary().then(setData).catch(console.error);
  }, []);

  return (
    <div className="flex flex-col gap-6">
      <FilterBar>
        <DateRangePicker />
      </FilterBar>
      <MetricGrid>
        <MetricCard title="Estimated Spend" value={data ? <CostDisplay amount={data.TotalCostMicros / 1000000} /> : "..."} />
        <MetricCard title="Cost / Request" value={data ? <CostDisplay amount={(data.TotalCostMicros / 1000000) / (data.Requests || 1)} /> : "..."} />
      </MetricGrid>
      <ChartCard title="Cost Trend">
        <div className="h-64 flex items-center justify-center text-muted-foreground border border-dashed rounded">Cost Chart Placeholder</div>
      </ChartCard>
    </div>
  );
}

export function ConversationsBody() {
  return (
    <div className="flex flex-col gap-6">
      <FilterBar>
        <SearchInput placeholder="Search conversations..." />
      </FilterBar>
      <div className="border border-line rounded-lg overflow-hidden bg-surface">
        <DataTable
          columns={[
            { key: "title", header: "Title", render: (r) => "Sample Conversation" },
            { key: "user", header: "User", render: (r) => "user@example.com" },
            { key: "model", header: "Model", render: (r) => <ModelBadge model="gpt-4" /> },
            { key: "privacy", header: "Privacy", render: (r) => <StatusBadge status="info" label="METADATA_ONLY" /> }
          ]}
          rows={[1, 2, 3]}
          isLoading={false}
          isError={false}
          error={null}
          empty="No conversations found."
          caption="Conversations"
        />
      </div>
    </div>
  );
}
""",
    'apps/web/src/lib/bodies-entities.tsx': """import { MetricGrid, MetricCard, DataTable, Avatar, UserPresence, ModelBadge } from "@/components/ui";
import { useEffect, useState } from "react";
import { listMembers, listProjects, listTeams, listModels } from "@/lib/control-api";

export function UsersBody() {
  const [users, setUsers] = useState<any[]>([]);
  useEffect(() => { listMembers().then(r => setUsers(r.items || [])); }, []);
  
  return (
    <div className="flex flex-col gap-6">
      <MetricGrid>
        <MetricCard title="Total Users" value={users.length} />
        <MetricCard title="Active Users" value={users.length} />
      </MetricGrid>
      <DataTable
        columns={[
          { key: "user", header: "User", render: (r: any) => <UserPresence name={r.user?.display_name || r.user?.email || "Unknown"} online={false} /> },
          { key: "role", header: "Role", render: (r: any) => r.role },
          { key: "joined", header: "Joined", render: (r: any) => new Date(r.member_since).toLocaleDateString() }
        ]}
        rows={users}
        isLoading={users.length === 0}
        isError={false}
        error={null}
        empty="No users found."
        caption="Users"
      />
    </div>
  );
}

export function TeamsObserveBody() {
  const [teams, setTeams] = useState<any[]>([]);
  useEffect(() => { listTeams().then(r => setTeams(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[
          { key: "name", header: "Team Name", render: (r: any) => r.name },
          { key: "desc", header: "Description", render: (r: any) => r.description }
        ]}
        rows={teams}
        isLoading={teams.length === 0}
        isError={false}
        error={null}
        empty="No teams found."
        caption="Teams"
      />
    </div>
  );
}

export function ProjectsObserveBody() {
  const [projects, setProjects] = useState<any[]>([]);
  useEffect(() => { listProjects().then(r => setProjects(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[
          { key: "name", header: "Project", render: (r: any) => r.name },
          { key: "slug", header: "Slug", render: (r: any) => <code className="text-xs">{r.slug}</code> },
          { key: "status", header: "Status", render: (r: any) => r.status }
        ]}
        rows={projects}
        isLoading={projects.length === 0}
        isError={false}
        error={null}
        empty="No projects found."
        caption="Projects"
      />
    </div>
  );
}

export function ModelsObserveBody() {
  const [models, setModels] = useState<any[]>([]);
  useEffect(() => { listModels().then(r => setModels(r.items || [])); }, []);
  return (
    <div className="flex flex-col gap-6">
      <DataTable
        columns={[
          { key: "name", header: "Model", render: (r: any) => <ModelBadge model={r.display_name || r.model_slug} /> },
          { key: "provider", header: "Provider", render: (r: any) => r.provider_kind },
          { key: "status", header: "Status", render: (r: any) => r.is_active ? "Active" : "Inactive" }
        ]}
        rows={models}
        isLoading={false}
        isError={false}
        error={null}
        empty="No models found."
        caption="Models"
      />
    </div>
  );
}
"""
}

for path, content in files.items():
    with open(path, 'w') as f:
        f.write(content)

print("Updated observe and entities bodies.")
