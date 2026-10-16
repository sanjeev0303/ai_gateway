import { Metadata } from "next";
import { PageShell, StatCard } from "@/components/dashboard";
import { getApiClient } from "@/lib/session";
import { notFound } from "next/navigation";
import { formatCompact, formatUsd, formatDate } from "@/lib/format";

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params;
  return { title: `Conversation ${id} · Sentinel` };
}

export default async function ConversationDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const { conversations, members } = await getApiClient();

  try {
    const [conversation, messages, membersList] = await Promise.all([
      conversations.get(id),
      conversations.messages(id),
      members.list().catch(() => ({ items: [] })),
    ]);

    const member = membersList.items.find((m) => m.user.id === conversation.user_id);
    const userName = member ? member.user.display_name || member.user.email : conversation.user_id;

    return (
      <div className="p-4 sm:p-6 space-y-6">
        <PageShell title={conversation.title || `Conversation ${id}`} description={`Detailed conversation trace.`}>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4 mb-6">
          <StatCard label="User" value={userName} />
          <StatCard label="Provider" value={conversation.provider ?? "—"} />
          <StatCard label="Model" value={conversation.model ?? "—"} />
          <StatCard label="Tokens" value={formatCompact(conversation.total_tokens)} />
          <StatCard label="Messages" value={formatCompact(conversation.message_count)} />
          <StatCard label="Cost" value={conversation.cost_micros ? formatUsd(conversation.cost_micros / 1000000) : "n/a"} />
          <StatCard label="Privacy State" value={conversation.privacy_state} />
          <StatCard label="Started At" value={formatDate(conversation.started_at)} />
        </div>

        <div className="space-y-4">
          <h2 className="text-lg font-semibold tracking-tight">Messages</h2>
          {messages.items.length === 0 ? (
            <p className="text-muted-foreground">No messages recorded or they were fully redacted.</p>
          ) : (
            <div className="space-y-4">
              {messages.items.map((msg) => (
                <div key={msg.id} className="p-4 rounded-xl border bg-card text-card-foreground shadow-sm">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="font-semibold capitalize text-sm">{msg.role}</span>
                    <span className="text-muted-foreground text-xs">{msg.tokens} tokens</span>
                  </div>
                  {msg.content ? (
                    <pre className="text-sm whitespace-pre-wrap font-sans text-muted-foreground bg-muted p-3 rounded-md">
                      {msg.content}
                    </pre>
                  ) : (
                    <p className="text-sm italic text-muted-foreground">Content redacted by privacy mode</p>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </PageShell>
    </div>
    );
  } catch (err) {
    return notFound();
  }
}
