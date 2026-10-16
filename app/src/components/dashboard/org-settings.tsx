"use client";

import * as React from "react";

import { StatCard, StatusBadge } from "@/components/dashboard";
import {
  updateOrganization,
  type OrganizationSettingsState,
} from "@/components/dashboard/org-settings-actions";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { formatDate } from "@/lib/format";
import { ORG_STATUS_TONE, type Organization } from "@/types";

function Field({
  label,
  hint,
  htmlFor,
  children,
}: {
  label: string;
  hint?: string;
  htmlFor: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={htmlFor} className="text-sm font-medium">
        {label}
      </label>
      {children}
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  );
}

export function OrgSettings({ org }: { org: Organization }) {
  const [name, setName] = React.useState(org.name);
  const [slug, setSlug] = React.useState(org.slug);
  const [privacyDefault, setPrivacyDefault] = React.useState(org.privacyDefault ?? "allow_all");
  const [retentionConversations, setRetentionConversations] = React.useState(String(org.retentionConversationsDays ?? 30));
  const [retentionUsage, setRetentionUsage] = React.useState(String(org.retentionUsageDays ?? 90));
  const [retentionAudits, setRetentionAudits] = React.useState(String(org.retentionAuditsDays ?? 365));

  const [state, formAction, pending] = React.useActionState<
    OrganizationSettingsState,
    FormData
  >(updateOrganization, {});

  const dirty =
    name !== org.name ||
    slug !== org.slug ||
    privacyDefault !== (org.privacyDefault ?? "allow_all") ||
    retentionConversations !== String(org.retentionConversationsDays ?? 30) ||
    retentionUsage !== String(org.retentionUsageDays ?? 90) ||
    retentionAudits !== String(org.retentionAuditsDays ?? 365);

  return (
    <div className="space-y-6">
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label="Member since"
          value={formatDate(org.createdAt)}
          hint={`Last updated ${formatDate(org.updatedAt)}`}
        />
        <StatCard label="Status" value={org.status} tone="info" />
        <StatCard
          label="Conversation Retention"
          value={`${retentionConversations} days`}
          tone="neutral"
          hint="Automated worker purge window"
        />
        <StatCard
          label="Audit Retention"
          value={`${retentionAudits} days`}
          tone="neutral"
          hint="Compliance log threshold"
        />
      </section>

      <form action={formAction} className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>General</CardTitle>
            <CardDescription>
              The name and slug appear in navigation, audit entries, and outbound email.
            </CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-2">
            <Field label="Organization name" htmlFor="org-name">
              <Input
                id="org-name"
                name="name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
              />
            </Field>
            <Field label="URL slug" htmlFor="org-slug">
              <Input
                id="org-slug"
                name="slug"
                value={slug}
                onChange={(event) => setSlug(event.target.value)}
                required
              />
            </Field>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Privacy & Data Retention</CardTitle>
            <CardDescription>
              Governance thresholds for background payload purging and default prompt privacy policies.
            </CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-2">
            <Field
              label="Default Privacy Mode"
              htmlFor="org-privacy-default"
              hint="Applied to incoming gateway requests when not overridden by policy"
            >
              <select
                id="org-privacy-default"
                name="privacy_default"
                value={privacyDefault}
                onChange={(e) => setPrivacyDefault(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <option value="METADATA_ONLY">METADATA_ONLY — Redact prompt and response contents</option>
                <option value="PROMPT_ONLY">PROMPT_ONLY — Record prompts only, withhold outputs</option>
                <option value="FULL_CONTENT">FULL_CONTENT — Store raw turns verbatim</option>
                <option value="OFF">OFF — Disable privacy filtering</option>
              </select>
            </Field>
            <Field
              label="Conversation History Retention (days)"
              htmlFor="retention-conversations"
              hint="Raw prompt and response bodies purged after this age"
            >
              <Input
                id="retention-conversations"
                name="retention_conversations_days"
                type="number"
                min={1}
                max={3650}
                value={retentionConversations}
                onChange={(e) => setRetentionConversations(e.target.value)}
              />
            </Field>
            <Field
              label="Usage Records Retention (days)"
              htmlFor="retention-usage"
              hint="Detailed per-request telemetry before rollup compaction"
            >
              <Input
                id="retention-usage"
                name="retention_usage_days"
                type="number"
                min={1}
                max={3650}
                value={retentionUsage}
                onChange={(e) => setRetentionUsage(e.target.value)}
              />
            </Field>
            <Field
              label="Audit Logs Retention (days)"
              htmlFor="retention-audits"
              hint="Immutable administrative audit trail preservation threshold"
            >
              <Input
                id="retention-audits"
                name="retention_audits_days"
                type="number"
                min={1}
                max={3650}
                value={retentionAudits}
                onChange={(e) => setRetentionAudits(e.target.value)}
              />
            </Field>
          </CardContent>
          <CardFooter className="gap-2">
            <Button size="sm" type="submit" disabled={!dirty || pending}>
              {pending ? "Saving…" : "Save changes"}
            </Button>
            {state.error && (
              <StatusBadge tone="danger">{state.error}</StatusBadge>
            )}
            {state.saved && !dirty && (
              <StatusBadge tone="success">Saved</StatusBadge>
            )}
          </CardFooter>
        </Card>
      </form>
    </div>
  );
}

export function OrgSettingsStatus({ org }: { org: Organization }) {
  return <StatusBadge tone={ORG_STATUS_TONE[org.status]}>{org.status}</StatusBadge>;
}