import { Metadata } from "next";
import { notFound } from "next/navigation";
import type { ConversationMessage, TranscriptState } from "@ai-governance/api-client";
import { PageShell, StatCard } from "@/components/dashboard";
import { getApiClient, requireSession } from "@/lib/session";
import { formatCompact, formatDate, formatDateTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import { DeleteConversationButton } from "./delete-button";
import { MessageMarkdown } from "./message-markdown";

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params;
  return { title: `Conversation ${id} · Tollseal` };
}

/** Shown when the gateway no longer holds the text. */
function ExpiredNotice() {
  return (
    <div role="status" className="rounded-xl border border-border bg-muted/50 px-4 py-3 text-sm">
      <p className="font-medium">Transcript no longer available</p>
      <p className="mt-1 text-muted-foreground">
        Tollseal never stores conversation text. The gateway keeps it in memory until the conversation is deleted,
        so it was cleared when the gateway last restarted. Counts and times remain.
      </p>
    </div>
  );
}

async function load(id: string) {
  const { conversations, members } = await getApiClient();
  try {
    return await Promise.all([
      conversations.get(id),
      conversations.messages(id),
      members.list().catch(() => ({ items: [] })),
    ]);
  } catch {
    return null;
  }
}

export default async function ConversationDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [loaded, { claims }] = await Promise.all([load(id), requireSession()]);
  if (!loaded) notFound();
  const [conversation, messages, membersList] = loaded;
  const canDelete = claims.role === "owner" || claims.role === "admin";

  const member = membersList.items.find((m) => m.user.id === conversation.user_id);
  const userName = member ? member.user.display_name || member.user.email : "Employee";
  const state: TranscriptState =
    messages.transcript ?? (messages.items.some((m) => m.content) ? "available" : "expired");

  const speaker = (msg: ConversationMessage) => {
    if (msg.role === "user") return userName;
    if (msg.role === "assistant") return conversation.model ?? "Assistant";
    if (msg.role === "tool") return msg.tool_name ?? "Tool";
    return "System";
  };

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title={conversation.title || "Conversation"}
        description={`${userName} · started ${formatDate(conversation.started_at)}`}
        actions={canDelete ? <DeleteConversationButton id={conversation.id} /> : undefined}
      >
        <div className="grid gap-3 sm:grid-cols-3 sm:gap-4">
          <StatCard label="Model" value={conversation.model ?? "—"} hint={conversation.provider ?? undefined} />
          <StatCard label="Messages" value={formatCompact(conversation.message_count)} />
          <StatCard label="Tokens" value={formatCompact(conversation.total_tokens)} />
        </div>

        <section aria-labelledby="transcript-heading" className="flex flex-col gap-4">
          <h2 id="transcript-heading" className="text-lg font-semibold tracking-tight">
            Transcript
          </h2>
          {state !== "available" && <ExpiredNotice />}

          {messages.items.length === 0 ? (
            <p className="text-sm text-muted-foreground">No messages in this conversation.</p>
          ) : state === "available" ? (
            <ol className="flex flex-col gap-5">
              {messages.items.map((msg) => {
                const fromUser = msg.role === "user";
                return (
                  <li key={msg.id} className={cn("flex flex-col gap-1.5", fromUser ? "items-end" : "items-start")}>
                    <p className="flex items-baseline gap-2 text-xs text-muted-foreground">
                      <span className="font-medium text-foreground">{speaker(msg)}</span>
                      <time dateTime={msg.created_at}>{formatDateTime(msg.created_at)}</time>
                      <span className="tabular-nums">{msg.tokens} tokens</span>
                    </p>
                    {msg.content ? (
                      <div
                        className={cn(
                          "min-w-0 rounded-2xl border px-4 py-3 text-sm leading-relaxed break-words",
                          fromUser
                            ? "max-w-[min(48rem,92%)] rounded-tr-md border-primary/20 bg-primary/10 whitespace-pre-wrap"
                            : "max-w-[min(52rem,96%)] rounded-tl-md border-border bg-card",
                        )}
                      >
                        {fromUser ? msg.content : <MessageMarkdown text={msg.content} />}
                      </div>
                    ) : (
                      <p className="text-sm text-muted-foreground italic">Text no longer available</p>
                    )}
                  </li>
                );
              })}
            </ol>
          ) : (
            <ol className="divide-y divide-border rounded-xl border border-border bg-card">
              {messages.items.map((msg) => (
                <li key={msg.id} className="flex items-baseline justify-between gap-4 px-4 py-2.5 text-sm">
                  <span className="min-w-0 truncate font-medium">{speaker(msg)}</span>
                  <span className="flex shrink-0 gap-4 text-xs text-muted-foreground tabular-nums">
                    <span>{msg.tokens} tokens</span>
                    <time dateTime={msg.created_at}>{formatDateTime(msg.created_at)}</time>
                  </span>
                </li>
              ))}
            </ol>
          )}
        </section>
      </PageShell>
    </div>
  );
}
