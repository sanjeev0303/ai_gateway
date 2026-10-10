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
import { formatNumber, formatRelative } from "@/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Conversations · Sentinel",
  description:
    "Recorded chat sessions, with message counts, token totals, and the privacy mode each one resolves to.",
};

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

/**
 * Privacy modes form a redaction ladder: the more turns a mode returns, the
 * more is exposed, so the strictest mode is the reassuring one. OFF and
 * FULL_CONTENT both return every turn, so they share the warning tone.
 */
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
  {
    key: "title",
    header: "Conversation",
    primary: "title",
    secondary: "routing",
    sort: "title",
    href: "detailUrl",
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
    key: "requests",
    header: "Requests",
    primary: "requests",
    format: "number",
    sort: "requests",
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
  {
    key: "privacy",
    header: "Privacy",
    primary: "privacy",
    tone: "privacyTone",
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
    team_id: typeof params.team_id === "string" ? params.team_id : undefined,
    provider: typeof params.provider === "string" ? params.provider : undefined,
    model: typeof params.model === "string" ? params.model : undefined,
    limit: typeof params.limit === "string" ? parseInt(params.limit, 10) : 50,
    offset: typeof params.offset === "string" ? parseInt(params.offset, 10) : 0,
  };

  const { conversations } = await getApiClient();
  const { items, has_more } = await conversations.list(filters);

  const totalMessages = items.reduce((sum, c) => sum + c.message_count, 0);
  const totalRequests = items.reduce((sum, c) => sum + c.request_count, 0);
  const totalTokens = items.reduce((sum, c) => sum + c.total_tokens, 0);

  const rows: ConversationRow[] = items.map((c) => ({
    id: c.id,
    title: c.title || "Untitled",
    user: c.user_id,
    routing: [c.provider, c.model].filter(Boolean).join(" · ") || "Provider not recorded",
    messages: c.message_count,
    requests: c.request_count,
    tokens: c.total_tokens,
    lastActivity: formatRelative(c.last_activity_at ?? c.started_at),
    detailUrl: `/app/observe/conversations/${c.id}`,
    ...privacyBadge(c.privacy_state),
  }));

  const limit = filters.limit ?? 50;
  const offset = filters.offset ?? 0;
  const currentPage = Math.floor(offset / limit);

  const buildPageUrl = (newOffset: number) => {
    const p = new URLSearchParams();
    if (filters.project_id) p.set("project_id", filters.project_id);
    if (filters.user_id) p.set("user_id", filters.user_id);
    if (filters.team_id) p.set("team_id", filters.team_id);
    if (filters.provider) p.set("provider", filters.provider);
    if (filters.model) p.set("model", filters.model);
    p.set("limit", limit.toString());
    p.set("offset", newOffset.toString());
    return `/app/observe/conversations?${p.toString()}`;
  };

  const nextHref = buildPageUrl(offset + limit);
  const prevHref = buildPageUrl(Math.max(0, offset - limit));

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Conversations"
        description="The sessions behind the token spend — how many turns each one holds, and how much of its content the privacy mode lets you read."
      >
        <ObserveFilters />
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Conversations"
            value={formatNumber(rows.length)}
            hint="First page of results"
          />
          <StatCard
            label="Messages"
            value={formatNumber(totalMessages)}
            hint="Across the page"
          />
          <StatCard
            label="Requests"
            value={formatNumber(totalRequests)}
            hint="Assistant turns only"
          />
          <StatCard label="Tokens" value={formatNumber(totalTokens)} hint="Across the page" />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Conversations</CardTitle>
            <CardDescription>
              Users are identified by ID. Requests counts assistant turns, so it runs lower than the
              message count. Per-conversation spend is absent because{" "}
              <code className="font-mono text-xs">conversation_messages</code> carries no link to{" "}
              <code className="font-mono text-xs">llm_requests</code> — the API returns{" "}
              <code className="font-mono text-xs">cost_micros</code> as null until that link exists.
              Transcript content is served per conversation by{" "}
              <code className="font-mono text-xs">/v1/conversations/&#123;id&#125;/messages</code> and
              is redacted server-side to the privacy mode shown here.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={conversationColumns}
              rows={rows}
              rowKey="id"
              pageSize={limit}
              emptyMessage="No conversations recorded."
              serverPaginated
              hasMore={has_more}
              currentPage={currentPage}
              nextHref={nextHref}
              prevHref={prevHref}
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}