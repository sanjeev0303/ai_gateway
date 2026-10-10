import { Metadata } from "next";
import { DataTable, PageShell, StatCard, type Column } from "@/components/dashboard";
import { getApiClient } from "@/lib/session";
import { notFound } from "next/navigation";
import { formatCompact, formatUsd, formatNumber, formatMs, formatDate, formatRelative } from "@/lib/format";
import type { StatusTone } from "@/types";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

interface UsageRow {
  id: string;
  name: string;
  href: string;
  calls: number;
  tokens: number;
  cost: number;
}

const createColumns = (header: string): readonly Column<UsageRow>[] => [
  { key: "name", header, primary: "name", sort: "name", href: "href" },
  { key: "calls", header: "Requests", primary: "calls", format: "compact", align: "right", sort: "calls" },
  { key: "tokens", header: "Tokens", primary: "tokens", format: "compact", align: "right", sort: "tokens" },
  { key: "cost", header: "Spend", primary: "cost", format: "usd", align: "right", sort: "cost" },
];

const modelColumns = createColumns("Model");
const providerColumns = createColumns("Provider");
const projectColumns = createColumns("Project");
const teamColumns = createColumns("Team");

interface ConversationRow {
  id: string;
  title: string;
  user: string;
  routing: string;
  messages: number;
  requests: number;
  tokens: number;
  lastActivity: string;
  privacy: string;
  privacyTone: StatusTone;
  detailUrl: string;
}

const PRIVACY: Record<string, { privacy: string; privacyTone: StatusTone }> = {
  METADATA_ONLY: { privacy: "Metadata only", privacyTone: "success" },
  PROMPT_ONLY: { privacy: "Prompt only", privacyTone: "info" },
  FULL_CONTENT: { privacy: "Full content", privacyTone: "warning" },
  OFF: { privacy: "No redaction", privacyTone: "warning" },
};

function privacyBadge(state: string): { privacy: string; privacyTone: StatusTone } {
  return PRIVACY[state] ?? { privacy: state, privacyTone: "neutral" };
}

const conversationColumns: readonly Column<ConversationRow>[] = [
  { key: "title", header: "Conversation", primary: "title", secondary: "routing", sort: "title", href: "detailUrl" },
  { key: "messages", header: "Messages", primary: "messages", format: "number", sort: "messages", align: "right" },
  { key: "requests", header: "Requests", primary: "requests", format: "number", sort: "requests", align: "right" },
  { key: "tokens", header: "Tokens", primary: "tokens", format: "compact", sort: "tokens", align: "right" },
  { key: "lastActivity", header: "Last activity", primary: "lastActivity", sort: "lastActivity" },
  { key: "privacy", header: "Privacy", primary: "privacy", tone: "privacyTone", sort: "privacy" },
];

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params;
  return { title: `User ${id} · Sentinel` };
}

export default async function UserDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const { members, usage, projects, teams, conversations } = await getApiClient();

  try {
    const [
      memberList,
      usageSummary,
      byModel,
      byProvider,
      byProject,
      byTeam,
      projectList,
      teamList,
      convs,
    ] = await Promise.all([
      members.list(),
      usage.summary({ user_id: id }),
      usage.byModel({ user_id: id }),
      usage.byProvider({ user_id: id }),
      usage.byProject({ user_id: id }),
      usage.byTeam({ user_id: id }),
      projects.list(),
      teams.list(),
      conversations.list({ user_id: id, limit: 10 }),
    ]);
    
    const user = memberList.items.find(m => m.user.id === id);
    if (!user) return notFound();

    const projectNames = new Map(projectList.items.map((p) => [p.id, p.name]));
    const teamNames = new Map(teamList.items.map((t) => [t.id, t.name]));

    const modelRows: UsageRow[] = byModel.items.map((row) => ({
      id: row.Model,
      name: row.Model,
      href: `/app/observe/usage?model=${encodeURIComponent(row.Model)}&user_id=${id}`,
      calls: row.Requests,
      tokens: row.TotalTokens,
      cost: (row.CostMicros ?? 0) / 1000000,
    }));

    const providerRows: UsageRow[] = byProvider.items.map((row) => ({
      id: row.Provider,
      name: row.Provider,
      href: `/app/observe/usage?provider=${encodeURIComponent(row.Provider)}&user_id=${id}`,
      calls: row.Requests,
      tokens: row.TotalTokens,
      cost: (row.CostMicros ?? 0) / 1000000,
    }));

    const projectRows: UsageRow[] = byProject.items.map((row) => ({
      id: row.ProjectID,
      name: projectNames.get(row.ProjectID) ?? row.ProjectID,
      href: `/app/observe/projects/${encodeURIComponent(row.ProjectID)}`,
      calls: row.Requests,
      tokens: row.TotalTokens,
      cost: (row.CostMicros ?? 0) / 1000000,
    }));

    const teamRows: UsageRow[] = byTeam.items.map((row) => ({
      id: row.TeamID,
      name: teamNames.get(row.TeamID) ?? row.TeamID,
      href: `/app/observe/teams/${encodeURIComponent(row.TeamID)}`,
      calls: row.Requests,
      tokens: row.TotalTokens,
      cost: (row.CostMicros ?? 0) / 1000000,
    }));

    const now = new Date();
    const convRows: ConversationRow[] = convs.items.map((c) => {
      const p = privacyBadge(c.privacy_state);
      return {
        id: c.id,
        title: c.title || "Untitled",
        user: c.user_id,
        routing: c.model ? `${c.provider} · ${c.model}` : "Unknown route",
        messages: c.message_count,
        requests: c.request_count,
        tokens: c.total_tokens,
        lastActivity: c.last_activity_at ? formatRelative(c.last_activity_at, now) : "Never",
        privacy: p.privacy,
        privacyTone: p.privacyTone,
        detailUrl: `/app/observe/conversations/${encodeURIComponent(c.id)}`,
      };
    });

    return (
      <div className="p-4 sm:p-6 space-y-6">
        <PageShell title={user.user.display_name || user.user.email} description={`User profile and usage.`}>
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4 mb-6">
            <StatCard label="Spend 30d" value={formatUsd((usageSummary.TotalCostMicros ?? 0) / 1000000)} />
            <StatCard label="Requests 30d" value={formatCompact(usageSummary.Requests ?? 0)} />
            <StatCard label="Tokens 30d" value={formatCompact(usageSummary.TotalTokens ?? 0)} />
            <StatCard label="Role" value={user.role} />
            <StatCard label="Errors 30d" value={formatCompact(usageSummary.Errors ?? 0)} />
            <StatCard label="Avg Latency" value={usageSummary.AvgLatencyMs ? formatMs(usageSummary.AvgLatencyMs) : "n/a"} />
            <StatCard label="Member Since" value={formatDate(user.member_since)} />
          </div>

          <div className="grid gap-4 lg:grid-cols-2 mb-6">
            <Card>
              <CardHeader>
                <CardTitle>Teams</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable columns={teamColumns} rows={teamRows} rowKey="id" pageSize={5} emptyMessage="No teams." />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Projects</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable columns={projectColumns} rows={projectRows} rowKey="id" pageSize={5} emptyMessage="No projects." />
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Models</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable columns={modelColumns} rows={modelRows} rowKey="id" pageSize={5} emptyMessage="No models." />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Providers</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable columns={providerColumns} rows={providerRows} rowKey="id" pageSize={5} emptyMessage="No providers." />
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Recent Activity (Conversations)</CardTitle>
            </CardHeader>
            <CardContent>
              <DataTable columns={conversationColumns} rows={convRows} rowKey="id" pageSize={10} emptyMessage="No recent activity." />
            </CardContent>
          </Card>
        </PageShell>
      </div>
    );
  } catch (err) {
    return notFound();
  }
}
