import type { Metadata } from "next";

import {
  DataTable,
  ObserveFilters,
  PageShell,
  StatCard,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatNumber, formatRelative } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import { HistoryPicker } from "./history-picker";
import { HistoryRail, type RailEmployee } from "./history-rail";

export const metadata: Metadata = {
  title: "Conversations · Tollseal",
  description:
    "Each employee's conversation history, by the gateway key it came through.",
};

interface ConversationRow {
  id: string;
  title: string;
  user: string;
  routing: string;
  messages: number;
  tokens: number;
  lastActivity: string;
  detailUrl: string;
}

const conversationColumns: readonly Column<ConversationRow>[] = [
  {
    key: "title",
    header: "Conversation",
    primary: "title",
    secondary: "routing",
    sort: "title",
    href: "detailUrl",
    grow: true,
  },
  { key: "user", header: "User", primary: "user", sort: "user" },
  {
    key: "messages",
    header: "Messages",
    primary: "messages",
    format: "number",
    sort: "messages",
    align: "right",
  },
  {
    key: "tokens",
    header: "Tokens",
    primary: "tokens",
    format: "compact",
    sort: "tokens",
    align: "right",
  },
  {
    key: "lastActivity",
    header: "Last activity",
    primary: "lastActivity",
    sort: "lastActivity",
  },
];

export default async function ConversationsPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const filters = {
    project_id: typeof params.project_id === "string" ? params.project_id : undefined,
    user_id: typeof params.user_id === "string" ? params.user_id : undefined,
    credential_id: typeof params.credential_id === "string" ? params.credential_id : undefined,
    team_id: typeof params.team_id === "string" ? params.team_id : undefined,
    provider: typeof params.provider === "string" ? params.provider : undefined,
    model: typeof params.model === "string" ? params.model : undefined,
    limit: typeof params.limit === "string" ? parseInt(params.limit, 10) : 50,
    offset: typeof params.offset === "string" ? parseInt(params.offset, 10) : 0,
  };

  const { conversations, members, credentials } = await getApiClient();
  const [{ items, has_more }, membersList, keyCounts, keyList] = await Promise.all([
    conversations.list(filters),
    members.list().catch(() => ({ items: [] })),
    conversations.byKey().catch(() => ({ items: [] })),
    credentials.listGateway().catch(() => ({ items: [] })),
  ]);

  const userMap = new Map<string, string>();
  for (const m of membersList.items) {
    userMap.set(m.user.id, m.user.display_name || m.user.email);
  }
  const keyMap = new Map(keyList.items.map((k) => [k.id, k]));

  // The rail: employees by latest activity, each with the keys their conversations came through.
  const employees: RailEmployee[] = [];
  const latest = new Map<string, string>();
  for (const row of keyCounts.items) {
    let employee = employees.find((e) => e.id === row.user_id);
    if (!employee) {
      employee = { id: row.user_id, name: userMap.get(row.user_id) ?? "Former member", total: 0, keys: [] };
      employees.push(employee);
    }
    employee.total += row.conversations;
    if ((latest.get(row.user_id) ?? "") < row.last_activity_at) latest.set(row.user_id, row.last_activity_at);
    const key = row.credential_id ? keyMap.get(row.credential_id) : undefined;
    employee.keys.push({
      id: row.credential_id,
      label: row.credential_id ? keyLabel(key?.name, employee.name) : "Key not recorded",
      prefix: key?.key_prefix,
      inactive: key && key.status !== "active" ? key.status : undefined,
      count: row.conversations,
    });
  }
  employees.sort((a, b) => (latest.get(b.id) ?? "").localeCompare(latest.get(a.id) ?? ""));

  const selectedEmployee = employees.find((e) => e.id === filters.user_id);
  const selectedKey = selectedEmployee?.keys.find((k) => (k.id ?? "none") === filters.credential_id);
  const heading = selectedKey
    ? `${selectedEmployee?.name} · ${selectedKey.label}`
    : selectedEmployee?.name ?? "All employees";
  const selectedCount = selectedKey?.count ?? selectedEmployee?.total ?? employees.reduce((sum, e) => sum + e.total, 0);

  const totalMessages = items.reduce((sum, c) => sum + c.message_count, 0);
  const totalTokens = items.reduce((sum, c) => sum + c.total_tokens, 0);

  const rows: ConversationRow[] = items.map((c) => {
    const key = c.credential_id ? keyMap.get(c.credential_id) : undefined;
    return {
      id: c.id,
      title: c.title || "Transcript expired",
      user: userMap.get(c.user_id) ?? c.user_id,
      routing: [c.provider, c.model, key?.key_prefix].filter(Boolean).join(" · ") || "Provider not recorded",
      messages: c.message_count,
      tokens: c.total_tokens,
      lastActivity: formatRelative(c.last_activity_at ?? c.started_at),
      detailUrl: `/app/observe/conversations/${c.id}`,
    };
  });
  // With one employee chosen, the User column only repeats the heading.
  const columns = filters.user_id ? conversationColumns.filter((c) => c.key !== "user") : conversationColumns;

  const limit = filters.limit ?? 50;
  const offset = filters.offset ?? 0;
  const currentPage = Math.floor(offset / limit);

  const buildUrl = (overrides: { user_id?: string; credential_id?: string; offset?: number }) => {
    const p = new URLSearchParams();
    if (filters.project_id) p.set("project_id", filters.project_id);
    if (filters.team_id) p.set("team_id", filters.team_id);
    if (filters.provider) p.set("provider", filters.provider);
    if (filters.model) p.set("model", filters.model);
    if (overrides.user_id) p.set("user_id", overrides.user_id);
    if (overrides.credential_id) p.set("credential_id", overrides.credential_id);
    if (overrides.offset) {
      p.set("limit", limit.toString());
      p.set("offset", overrides.offset.toString());
    }
    const query = p.toString();
    return query ? `/app/observe/conversations?${query}` : "/app/observe/conversations";
  };
  const pageUrl = (newOffset: number) =>
    buildUrl({ user_id: filters.user_id, credential_id: filters.credential_id, offset: newOffset });
  const railHref = (userId?: string, keyId?: string | null) =>
    buildUrl({ user_id: userId, credential_id: keyId === undefined ? undefined : keyId ?? "none" });

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Conversations"
        description="Each employee's conversation history, by the gateway key it came through."
      >
        <ObserveFilters exclude={["status", "service_account_id", "user_id"]} />

        <div className="grid gap-5 xl:grid-cols-[17rem_minmax(0,1fr)]">
          <aside className="hidden xl:block">
            <div
              data-lenis-prevent
              className="sticky top-24 max-h-[calc(100dvh-8rem)] overflow-y-auto overscroll-contain rounded-2xl border border-border bg-card p-2"
            >
              <HistoryRail
                employees={employees}
                userId={filters.user_id}
                keyId={filters.credential_id}
                href={railHref}
              />
            </div>
          </aside>

          <div className="flex min-w-0 flex-col gap-5">
            <div className="xl:hidden">
              <HistoryPicker employees={employees} />
            </div>

            <section className="grid gap-3 sm:grid-cols-3 sm:gap-4">
              <StatCard label="Conversations" value={formatNumber(selectedCount)} />
              <StatCard label="Messages" value={formatNumber(totalMessages)} hint="On this page" />
              <StatCard label="Tokens" value={formatNumber(totalTokens)} hint="On this page" />
            </section>

            <Card>
              <CardHeader>
                <CardTitle>{heading}</CardTitle>
                <CardDescription>
                  Conversation text is never stored. The gateway holds it in memory until a conversation is
                  deleted; a gateway restart clears it, leaving only counts and times.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <DataTable
                  columns={columns}
                  rows={rows}
                  rowKey="id"
                  pageSize={limit}
                  emptyMessage="No conversations for this selection."
                  serverPaginated
                  hasMore={has_more}
                  currentPage={currentPage}
                  nextHref={pageUrl(offset + limit)}
                  prevHref={pageUrl(Math.max(0, offset - limit))}
                />
              </CardContent>
            </Card>
          </div>
        </div>
      </PageShell>
    </div>
  );
}

/** "Rahul Employee laptop key" under Rahul Employee reads as "Laptop key". */
function keyLabel(name: string | undefined, employeeName: string): string {
  if (!name) return "Removed key";
  const prefix = `${employeeName} `;
  if (name.toLowerCase().startsWith(prefix.toLowerCase()) && name.length > prefix.length) {
    const rest = name.slice(prefix.length);
    return rest.charAt(0).toUpperCase() + rest.slice(1);
  }
  return name;
}
