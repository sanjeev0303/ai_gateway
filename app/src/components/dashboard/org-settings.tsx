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
  const [state, formAction, pending] = React.useActionState<
    OrganizationSettingsState,
    FormData
  >(updateOrganization, {});

  const dirty = name !== org.name || slug !== org.slug;

  return (
    <div className="space-y-6">
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label="Member since"
          value={formatDate(org.createdAt)}
          hint={`Last updated ${formatDate(org.updatedAt)}`}
        />
        <StatCard label="Status" value={org.status} tone="info" />
      </section>

      <Card>
        <CardHeader>
          <CardTitle>General</CardTitle>
          <CardDescription>
            The name and slug appear in navigation, audit entries, and outbound
            email.
          </CardDescription>
        </CardHeader>
        <form action={formAction}>
          <CardContent className="grid gap-4 sm:grid-cols-2">
            <Field label="Organization name" htmlFor="org-name">
              <Input
                id="org-name"
                name="name"
                value={name}
                onChange={(event) => {
                  setName(event.target.value);
                }}
              />
            </Field>
            <Field label="URL slug" htmlFor="org-slug">
              <Input
                id="org-slug"
                name="slug"
                value={slug}
                onChange={(event) => {
                  setSlug(event.target.value);
                }}
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
        </form>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Not configurable yet</CardTitle>
          <CardDescription>
            Settings the console does not expose, because the control API has no
            endpoint for them.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-muted-foreground">
          <p>
            Name and slug are the only writable fields on an organization.
            <code className="mx-1 font-mono text-xs">PATCH /v1/organization</code>
            accepts nothing else, and the console only offers what that route can
            persist.
          </p>
          <p>
            There are no single sign-on, multi-factor, IP allow-list, data
            retention, or notification-routing settings — those subsystems do not
            exist yet. There is also no endpoint to suspend or delete an
            organization, so no such control is offered here.
          </p>
          <p>
            Subscription and billing fields exist on the response but are not
            editable here; this deployment does not meter or bill organizations.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

export function OrgSettingsStatus({ org }: { org: Organization }) {
  return <StatusBadge tone={ORG_STATUS_TONE[org.status]}>{org.status}</StatusBadge>;
}