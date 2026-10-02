import type { Metadata } from "next";

import { BarChart, DonutChart } from "@/components/charts";
import {
  DataTable,
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
import { Badge } from "@/components/ui/badge";
import {
  formatCompact,
  formatNumber,
  formatRelative,
  formatUsd,
} from "@/lib/format";
import { NOW, conversations, models } from "@/lib/mock-data";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Conversations · Sentinel",
  description: "Every gateway session with its turn count, token cost, and redactor verdict.",
};

const modalityById = new Map(models.map((model) => [model.id, model.modality]));

const MODALITY_TONE: Record<string, StatusTone> = {
  text: "info",
  vision: "success",
  audio: "warning",
  embedding: "neutral",
  multimodal: "danger",
};

const modalityOf = (modelId: string) => modalityById.get(modelId) ?? "text";

interface ConversationRow {
  id: string;
  title: string;
  user: string;
  project: string;
  model: string;
  modality: string;
  modalityTone: StatusTone;
  started: string;
  turns: string;
  turnCount: number;
  tokens: string;
  cost: string;
  costValue: number;
  review: string;
  reviewTone: StatusTone;
}

const conversationColumns: readonly Column<ConversationRow>[] = [
  {
    key: "title",
    header: "Conversation",
    primary: "title",
    secondary: "project",
    sort: "title",
  },
  { key: "user", header: "User", primary: "user", sort: "user" },
  { key: "model", header: "Model", primary: "model", sort: "model" },
  { key: "modality", header: "Modality", primary: "modality", tone: "modalityTone", sort: "modality" },
  { key: "started", header: "Started", primary: "started", sort: "started" },
  {
    key: "turns",
    header: "Turns",
    primary: "turns",
    align: "right",
    sort: "turnCount",
  },
  { key: "tokens", header: "Tokens", primary: "tokens", align: "right" },
  {
    key: "cost",
    header: "Cost",
    primary: "cost",
    format: "usd",
    align: "right",
    sort: "costValue",
  },
  { key: "review", header: "Review", primary: "review", tone: "reviewTone", sort: "review" },
];

const rows: ConversationRow[] = [...conversations]
  .sort((a, b) => (a.startedAt < b.startedAt ? 1 : -1))
  .map((conversation) => {
    const modality = modalityOf(conversation.modelId);
    return {
      id: conversation.id,
      title: conversation.title,
      user: conversation.userName,
      project: conversation.projectName,
      model: conversation.modelName,
      modality,
      modalityTone: MODALITY_TONE[modality] ?? "neutral",
      started: formatRelative(conversation.startedAt, NOW),
      turns: formatNumber(conversation.turnCount),
      turnCount: conversation.turnCount,
      tokens: formatCompact(conversation.inputTokens + conversation.outputTokens),
      cost: formatUsd(conversation.cost),
      costValue: conversation.cost,
      review: conversation.flagged ? "Flagged" : "Clear",
      reviewTone: conversation.flagged ? "danger" : "success",
    };
  });

const flagged = conversations.filter((conversation) => conversation.flagged);
const totalCost = conversations.reduce((acc, conversation) => acc + conversation.cost, 0);
const totalTokens = conversations.reduce(
  (acc, conversation) => acc + conversation.inputTokens + conversation.outputTokens,
  0,
);

const modalitySet = [...new Set(conversations.map((c) => modalityOf(c.modelId)))];
const modalityTotals = modalitySet.map((modality) => ({
  label: modality,
  value: conversations
    .filter((conversation) => modalityOf(conversation.modelId) === modality)
    .reduce((acc, conversation) => acc + conversation.cost, 0),
}));

const costliest = [...conversations]
  .sort((a, b) => b.cost - a.cost)
  .slice(0, 8)
  .map((conversation) => ({ label: conversation.title, value: conversation.cost }));

export default function ConversationsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Conversations"
        description="Every session the gateway proxied, with the token cost it incurred and whether the redactor held."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Conversations"
            value={formatNumber(conversations.length)}
            hint="Trailing 24 hours"
          />
          <StatCard
            label="Cost"
            value={formatUsd(totalCost)}
            hint={`${formatUsd(totalCost / Math.max(1, conversations.length))} average`}
          />
          <StatCard
            label="Tokens"
            value={formatCompact(totalTokens)}
            hint={`${formatCompact(conversations.reduce((acc, c) => acc + c.outputTokens, 0))} output`}
          />
          <StatCard
            label="Held for review"
            value={formatNumber(flagged.length)}
            tone={flagged.length > 0 ? "danger" : "success"}
            hint="Matched a detector"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Costliest conversations</CardTitle>
              <CardDescription>
                Long agentic runs dominate spend far more than single calls.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={costliest} formatValue={formatUsd} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by modality</CardTitle>
              <CardDescription>Audio and vision remain a rounding error.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={modalityTotals}
                centerLabel={formatCompact(totalCost)}
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>
        </section>

        {flagged.length > 0 ? (
          <Card>
            <CardHeader>
              <CardTitle>Redactor holds</CardTitle>
              <CardDescription>
                Content matched a PII or jailbreak detector. The turn text is
                withheld until a reviewer clears it.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {flagged.map((conversation) => (
                <div
                  key={conversation.id}
                  className="flex flex-col gap-2 rounded-lg border border-border p-3 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-foreground">
                      {conversation.title}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {conversation.flagReason}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <span className="text-xs text-muted-foreground">
                      {formatRelative(conversation.startedAt, NOW)}
                    </span>
                    <Badge variant="destructive">Held</Badge>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        ) : null}

        <Card>
          <CardHeader>
            <CardTitle>All conversations</CardTitle>
            <CardDescription>
              Newest first. Tool turns are counted but billed at zero.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={conversationColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No conversations recorded."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
