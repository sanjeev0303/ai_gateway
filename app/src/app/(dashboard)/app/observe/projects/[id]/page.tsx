import { Metadata } from "next";
import { DataTable, PageShell, StatCard, type Column } from "@/components/dashboard";
import { getApiClient } from "@/lib/session";
import { notFound } from "next/navigation";
import { formatCompact, formatUsd, formatDate, formatRelative } from "@/lib/format";
import type { StatusTone } from "@/types";
import {
  Card,
  CardContent,
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

interface MemberRow {
  id: string;
  name: string;
  email: string;
  role: string;
  href: string;
}

const memberColumns: readonly Column<MemberRow>[] = [
  { key: "name", header: "Member", primary: "name", secondary: "email", sort: "name", href: "href" },
  { key: "role", header: "Role", primary: "role", sort: "role" },
];

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

interface CredentialRow {
  id: string;
  name: string;
  scopes: string;
  expiresAt: string;
}

const credentialColumns: readonly Column<CredentialRow>[] = [
  { key: "name", header: "Name", primary: "name", sort: "name" },
  { key: "scopes", header: "Scopes", primary: "scopes", sort: "scopes" },
  { key: "expiresAt", header: "Expires", primary: "expiresAt", sort: "expiresAt" },
];

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params;
  return { title: `Project ${id} · Sentinel` };
}

export default async function ProjectDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const { projects, usage, conversations, credentials } = await getApiClient();

  try {
    const [
      project,
      usageSummary,
      membersList,
      byModel,
      byProvider,
      convs,
      allCreds,
    ] = await Promise.all([
      projects.get(id),
      usage.summary({ project_id: id }),
      projects.listMembers(id),
      usage.byModel({ project_id: id }),
      usage.byProvider({ project_id: id }),
      conversations.list({ project_id: id, limit: 10 }),
      credentials.listGateway(),
    ]);

    const memberRows: MemberRow[] = membersList.items.map((m) => ({
      id: m.user.id,
      name: m.user.display_name || m.user.email,
      email: m.user.email,
      role: m.role,
      href: `/app/observe/users/${encodeURIComponent(m.user.id)}`,
    }));

    const modelRows: UsageRow[] = byModel.items.map((row) => ({
      id: row.Model,
      name: row.Model,
      href: `/app/observe/usage?model=${encodeURIComponent(row.Model)}&project_id=${id}`,
      calls: row.Requests,
      tokens: row.TotalTokens,
      cost: (row.CostMicros ?? 0) / 1000000,
    }));

    const providerRows: UsageRow[] = byProvider.items.map((row) => ({
      id: row.Provider,
      name: row.Provider,
      href: `/app/observe/usage?provider=${encodeURIComponent(row.Provider)}&project_id=${id}`,
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

    const credRows: CredentialRow[] = allCreds.items
      .filter((c) => c.project_id === id)
      .map((c) => ({
        id: c.id,
        name: c.name,
        scopes: c.scopes.join(", "),
        expiresAt: c.expires_at ? formatDate(c.expires_at) : "Never",
      }));

    return (
      <div className="p-4 sm:p-6 space-y-6">
        <PageShell title={project.name} description={`Project profile and usage.`}>
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4 mb-6">
            <StatCard label="Spend 30d" value={formatUsd((usageSummary.TotalCostMicros ?? 0) / 1000000)} />
            <StatCard label="Requests 30d" value={formatCompact(usageSummary.Requests ?? 0)} />
            <StatCard label="Tokens 30d" value={formatCompact(usageSummary.TotalTokens ?? 0)} />
            <StatCard label="Errors 30d" value={formatCompact(usageSummary.Errors ?? 0)} />
          </div>

          <Card className="mb-6">
            <CardHeader>
              <CardTitle>Members</CardTitle>
            </CardHeader>
            <CardContent>
              <DataTable columns={memberColumns} rows={memberRows} rowKey="id" pageSize={5} emptyMessage="No members." />
            </CardContent>
          </Card>

          <div className="grid gap-4 lg:grid-cols-2 mb-6">
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

          <Card className="mb-6">
            <CardHeader>
              <CardTitle>Credentials</CardTitle>
            </CardHeader>
            <CardContent>
              <DataTable columns={credentialColumns} rows={credRows} rowKey="id" pageSize={5} emptyMessage="No gateway credentials." />
            </CardContent>
          </Card>

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
