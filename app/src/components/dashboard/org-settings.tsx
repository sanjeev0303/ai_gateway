"use client";

import * as React from "react";
import { Trash2 } from "lucide-react";

import { StatCard, StatusBadge } from "@/components/dashboard";
import { Badge } from "@/components/ui/badge";
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
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import { formatDate, formatUsd } from "@/lib/format";
import { ORG_STATUS_TONE, PLAN_LABELS, type Organization } from "@/types";

const RETENTION_OPTIONS = ["30", "90", "180", "400"] as const;

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

function Toggle({
  id,
  label,
  description,
  defaultChecked,
}: {
  id: string;
  label: string;
  description: string;
  defaultChecked: boolean;
}) {
  return (
    <div className="flex items-start justify-between gap-6 py-3">
      <div className="min-w-0 space-y-0.5">
        <label htmlFor={id} className="text-sm font-medium">
          {label}
        </label>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
      <button
        type="button"
        id={id}
        role="switch"
        aria-checked={defaultChecked}
        className={`relative h-5 w-9 shrink-0 rounded-full transition-colors ${
          defaultChecked ? "bg-primary" : "bg-input"
        }`}
      >
        <span
          className={`absolute top-0.5 size-4 rounded-full bg-background shadow transition-transform ${
            defaultChecked ? "translate-x-4.5" : "translate-x-0.5"
          }`}
        />
      </button>
    </div>
  );
}

export function OrgSettings({ org }: { org: Organization }) {
  const [name, setName] = React.useState(org.name);
  const [slug, setSlug] = React.useState(org.slug);
  const [billingEmail, setBillingEmail] = React.useState(org.billingEmail);
  const [retention, setRetention] = React.useState<string>(RETENTION_OPTIONS[3]);
  const [saved, setSaved] = React.useState(false);

  const seatPct = Math.round((org.seatsUsed / org.seatsTotal) * 100);
  const dirty =
    name !== org.name ||
    slug !== org.slug ||
    billingEmail !== org.billingEmail ||
    retention !== RETENTION_OPTIONS[3];

  return (
    <div className="space-y-6">
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard label="Plan" value={PLAN_LABELS[org.plan]} tone="info" hint="Enterprise agreement" />
        <StatCard
          label="Seats"
          value={`${org.seatsUsed} / ${org.seatsTotal}`}
          hint={`${seatPct}% licensed`}
        />
        <StatCard
          label="Monthly value"
          value={formatUsd(org.mrr)}
          hint="Annual recurring revenue"
        />
        <StatCard
          label="Member since"
          value={formatDate(org.createdAt)}
          hint={`Region ${org.region}`}
        />
      </section>

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
              value={name}
              onChange={(event) => {
                setName(event.target.value);
                setSaved(false);
              }}
            />
          </Field>
          <Field label="URL slug" htmlFor="org-slug" hint={`sentinel.dev/${slug}`}>
            <Input
              id="org-slug"
              value={slug}
              onChange={(event) => {
                setSlug(event.target.value);
                setSaved(false);
              }}
            />
          </Field>
          <Field
            label="Billing email"
            htmlFor="org-email"
            hint="Receives invoices and renewal notices."
          >
            <Input
              id="org-email"
              type="email"
              value={billingEmail}
              onChange={(event) => {
                setBillingEmail(event.target.value);
                setSaved(false);
              }}
            />
          </Field>
          <Field label="Data residency" htmlFor="org-region" hint="Request data is stored in this region.">
            <div className="flex h-8 items-center rounded-lg border px-2.5 text-sm">
              {org.region}
            </div>
          </Field>
        </CardContent>
        <CardFooter className="gap-2">
          <Button size="sm" disabled={!dirty} onClick={() => setSaved(true)}>
            Save changes
          </Button>
          {saved && (
            <StatusBadge tone="success">Saved</StatusBadge>
          )}
        </CardFooter>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Seats</CardTitle>
          <CardDescription>
            {org.seatsTotal - org.seatsUsed} of {org.seatsTotal} seats are unassigned.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <Progress value={seatPct} />
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="outline">
              Assign seats
            </Button>
            <Button size="sm" variant="ghost">
              Purchase more
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Security</CardTitle>
          <CardDescription>
            Applies organization-wide. Individual members can still narrow their own defaults.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <Field label="Default data retention" htmlFor="org-retention" hint="Applies to new projects; existing ones keep their own setting.">
              <div className="flex flex-wrap gap-2" id="org-retention">
                {RETENTION_OPTIONS.map((option) => (
                  <Button
                    key={option}
                    type="button"
                    size="sm"
                    variant={retention === option ? "default" : "outline"}
                    onClick={() => {
                      setRetention(option);
                      setSaved(false);
                    }}
                  >
                    {option} days
                  </Button>
                ))}
              </div>
            </Field>
          </div>
          <Separator />
          <div className="divide-y">
            <Toggle
              id="toggle-sso"
              label="Require single sign-on"
              description="Members must authenticate through the configured identity provider."
              defaultChecked
            />
            <Toggle
              id="toggle-mfa"
              label="Require multi-factor authentication"
              description="Enforced for members with administrative roles."
              defaultChecked
            />
            <Toggle
              id="toggle-ip"
              label="Restrict access by IP range"
              description="Only allow sign-in from allow-listed ranges."
              defaultChecked={false}
            />
            <Toggle
              id="toggle-export"
              label="Allow data export"
              description="Members may export traces and usage data to CSV."
              defaultChecked
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Notifications</CardTitle>
          <CardDescription>Where operational alerts are delivered.</CardDescription>
        </CardHeader>
        <CardContent className="divide-y">
          <Toggle
            id="notify-incident"
            label="Provider incidents"
            description="Provider status changes and elevated error rates."
            defaultChecked
          />
          <Toggle
            id="notify-budget"
            label="Budget thresholds"
            description="When a project crosses 80% and 100% of its budget."
            defaultChecked
          />
          <Toggle
            id="notify-policy"
            label="Policy drift"
            description="When a project falls out of compliance with a policy."
            defaultChecked={false}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Danger zone</CardTitle>
          <CardDescription>
            Destructive and immediate. There is no undo for either action.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-destructive/40 p-4">
            <div className="min-w-0">
              <p className="text-sm font-medium">Suspend organization</p>
              <p className="text-xs text-muted-foreground">
                Blocks all gateway traffic while keeping data intact.
              </p>
            </div>
            <Button size="sm" variant="outline">
              Suspend
            </Button>
          </div>
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-destructive/40 p-4">
            <div className="min-w-0">
              <p className="text-sm font-medium">Delete organization</p>
              <p className="text-xs text-muted-foreground">
                Permanently removes {org.name}, its projects, traces, and audit history.
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="outline">{org.slug}</Badge>
              <Button size="sm" variant="destructive">
                <Trash2 className="size-4" aria-hidden /> Delete
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export function OrgSettingsStatus({ org }: { org: Organization }) {
  return <StatusBadge tone={ORG_STATUS_TONE[org.status]}>{org.status}</StatusBadge>;
}
