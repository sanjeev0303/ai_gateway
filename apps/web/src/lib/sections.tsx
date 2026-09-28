"use client";

import { useEffect, useState, type FormEvent, type ReactNode } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import {
  DataTable,
  EmptyNote,
  ErrorNote,
  Field,
  Mono,
  SecretReveal,
  StatusPill,
  SubmitButton,
  formatDate,
  formatDateTime,
  never,
  type Column,
} from "@/components/ui";
import {
  ROLES,
  canManage,
  type ApiCredential,
  type GatewayCredential,
  type Invitation,
  type Member,
  type Model,
  type Project,
  type ProviderAccount,
  type ProviderCredential,
  type ProviderHealth,
  type ServiceAccount,
  type Team,
} from "@/lib/control-api";
import {
  useApiCredentials,
  useCreateApiCredential,
  useCreateGatewayCredential,
  useCreateProject,
  useCreateProviderAccount,
  useCreateProviderCredential,
  useCreateTeam,
  useExpireGatewayCredential,
  useGatewayCredentials,
  useInvite,
  useInvitations,
  useMembers,
  useModels,
  useProjects,
  useProviderAccounts,
  useProviderCredentials,
  useProviderHealth,
  useRemoveMember,
  useRestoreGatewayCredential,
  useRevokeApiCredential,
  useRevokeGatewayCredential,
  useRevokeInvitation,
  useRevokeProviderCredential,
  useRotateGatewayCredential,
  useRotateProviderCredential,
  useServiceAccounts,
  useSession,
  useSetMemberRole,
  useSyncModels,
  useTeams,
} from "@/lib/queries";

function Action({
  label,
  onClick,
  disabled,
  pending,
  tone: variant = "ghost",
}: {
  readonly label: string;
  readonly onClick: () => void;
  readonly disabled?: boolean;
  readonly pending?: boolean;
  readonly tone?: "ghost" | "danger";
}) {
  return (
    <button
      type="button"
      className={`btn btn-${variant}`}
      onClick={onClick}
      disabled={disabled || pending}
    >
      {pending ? "…" : label}
    </button>
  );
}

function Actions({ children }: { readonly children: ReactNode }) {
  return <div className="row-actions">{children}</div>;
}

function useRole() {
  return useSession().data?.claims.role ?? "";
}

function useManager() {
  return canManage(useRole());
}

const PROVIDER_KINDS = ["openai", "anthropic", "gemini", "azure_openai", "custom"] as const;
const CREDENTIAL_KINDS = ["api_key", "oauth"] as const;
const SCOPES = ["gateway", "control", "all"] as const;
const SUBJECT_TYPES = ["user", "service_account"] as const;

const nameSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
  description: z.string(),
});
type NameValues = z.infer<typeof nameSchema>;

const inviteSchema = z.object({
  email: z.string().trim().min(1, "Email is required"),
  role: z.string().min(1),
});
type InviteValues = z.infer<typeof inviteSchema>;

const apiCredentialFormSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
  scope: z.string().min(1),
  service_account_id: z.string(),
});
type ApiCredentialFormValues = z.infer<typeof apiCredentialFormSchema>;

const gatewayCredentialFormSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
  subject_type: z.enum(SUBJECT_TYPES),
  subject_id: z.string().min(1, "Pick a subject"),
  scopes: z.array(z.string()).min(1, "Pick at least one scope"),
  expires_at: z.string(),
});
type GatewayCredentialFormValues = z.infer<typeof gatewayCredentialFormSchema>;

const providerAccountFormSchema = z.object({
  provider_kind: z.string().min(1),
  name: z.string().trim().min(1, "Name is required"),
  base_url: z.string(),
});
type ProviderAccountFormValues = z.infer<typeof providerAccountFormSchema>;

const providerCredentialFormSchema = z.object({
  accountId: z.string().min(1, "Pick an account"),
  name: z.string().trim().min(1, "Name is required"),
  kind: z.string().min(1),
  secret: z.string().min(1, "Secret is required"),
});
type ProviderCredentialFormValues = z.infer<typeof providerCredentialFormSchema>;

const syncModelsSchema = z.object({
  providerKind: z.string().min(1),
  apiKey: z.string().min(1, "API key is required"),
});
type SyncModelsValues = z.infer<typeof syncModelsSchema>;

function NameForm({
  submitLabel,
  pending,
  error,
  onSubmit,
}: {
  readonly submitLabel: string;
  readonly pending: boolean;
  readonly error: Error | null;
  readonly onSubmit: (values: NameValues) => void;
}) {
  const form = useForm<NameValues>({
    resolver: zodResolver(nameSchema),
    defaultValues: { name: "", description: "" },
  });
  return (
    <form className="inline-form" onSubmit={form.handleSubmit(onSubmit)}>
      <div className="form-grid">
        <Field label="Name" error={form.formState.errors.name?.message}>
          <input className="input" {...form.register("name")} />
        </Field>
        <Field label="Description" error={form.formState.errors.description?.message}>
          <input className="input" {...form.register("description")} />
        </Field>
      </div>
      <SubmitButton pending={pending} label={submitLabel} pendingLabel="Creating…" />
      {error !== null && <ErrorNote error={error} />}
    </form>
  );
}

export function MembersBody() {
  const query = useMembers();
  const role = useSession().data?.claims.role ?? "";
  const setRole = useSetMemberRole();
  const remove = useRemoveMember();
  const manage = canManage(role);

  const columns: readonly Column<Member>[] = [
    { key: "name", header: "Name", render: (m) => m.user.display_name || m.user.email },
    { key: "email", header: "Email", render: (m) => <Mono>{m.user.email}</Mono> },
    { key: "status", header: "Status", render: (m) => <StatusPill value={m.user.status} /> },
    {
      key: "role",
      header: "Role",
      render: (m) =>
        manage ? (
          <select
            className="input input-inline"
            value={m.role}
            aria-label={`Role for ${m.user.email}`}
            disabled={setRole.isPending}
            onChange={(e) => setRole.mutate({ userId: m.user.id, role: e.target.value })}
          >
            {ROLES.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </select>
        ) : (
          <StatusPill value={m.role} />
        ),
    },
    { key: "since", header: "Member since", render: (m) => formatDate(m.member_since) },
    { key: "login", header: "Last login", render: (m) => never(m.user.last_login_at, "Never") },
    ...(manage
      ? [
          {
            key: "actions",
            header: "",
            align: "right" as const,
            render: (m: Member) => (
              <Actions>
                <Action
                  label="Remove"
                  tone="danger"
                  pending={remove.isPending}
                  onClick={() => remove.mutate(m.user.id)}
                />
              </Actions>
            ),
          },
        ]
      : []),
  ];

  return (
    <>
      <DataTable
        caption="Organization members"
        columns={columns}
        rows={query.data ?? []}
        isLoading={query.isPending}
        isError={query.isError}
        error={query.error}
        empty="No members yet."
      />
      {(setRole.isError || remove.isError) && (
        <ErrorNote error={setRole.error ?? remove.error} />
      )}
    </>
  );
}

export function InvitationsBody() {
  const query = useInvitations();
  const revoke = useRevokeInvitation();
  const invite = useInvite();
  const manage = useManager();
  const [open, setOpen] = useState(false);
  const form = useForm<InviteValues>({
    resolver: zodResolver(inviteSchema),
    defaultValues: { email: "", role: "member" },
  });
  const columns: readonly Column<Invitation>[] = [
    { key: "email", header: "Email", render: (i) => <Mono>{i.email}</Mono> },
    { key: "role", header: "Role", render: (i) => <StatusPill value={i.role} /> },
    { key: "status", header: "Status", render: (i) => <StatusPill value={i.status} /> },
    { key: "expires", header: "Expires", render: (i) => formatDateTime(i.expires_at) },
    { key: "created", header: "Invited", render: (i) => formatDate(i.created_at) },
    ...(manage
      ? [
          {
            key: "actions",
            header: "",
            align: "right" as const,
            render: (i: Invitation) => (
              <Actions>
                <Action
                  label="Revoke"
                  tone="danger"
                  pending={revoke.isPending}
                  onClick={() => revoke.mutate(i.id)}
                />
              </Actions>
            ),
          },
        ]
      : []),
  ];
  return (
    <>
      {manage && !open && (
        <div className="row-actions">
          <Action label="Invite member" onClick={() => setOpen(true)} />
        </div>
      )}
      {manage && open && (
        <form
          className="inline-form"
          onSubmit={form.handleSubmit((values) =>
            invite.mutate(values, {
              onSuccess: () => {
                form.reset();
                setOpen(false);
              },
            }),
          )}
        >
          <div className="form-grid">
            <Field label="Email" error={form.formState.errors.email?.message}>
              <input className="input" type="email" autoComplete="off" {...form.register("email")} />
            </Field>
            <Field label="Role">
              <select className="input" {...form.register("role")}>
                {ROLES.map((role) => (
                  <option key={role} value={role}>
                    {role}
                  </option>
                ))}
              </select>
            </Field>
          </div>
          <SubmitButton pending={invite.isPending} label="Send invitation" pendingLabel="Sending…" />
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => setOpen(false)}
            disabled={invite.isPending}
          >
            Cancel
          </button>
          {invite.error !== null && <ErrorNote error={invite.error} />}
        </form>
      )}
      <DataTable
        caption="Pending invitations"
        columns={columns}
        rows={query.data ?? []}
        isLoading={query.isPending}
        isError={query.isError}
        error={query.error}
        empty="No invitations."
      />
    </>
  );
}

export function ProjectsBody() {
  const query = useProjects();
  const create = useCreateProject();
  const manage = useManager();
  const [open, setOpen] = useState(false);
  const columns: readonly Column<Project>[] = [
    { key: "name", header: "Name", render: (p) => p.name },
    { key: "slug", header: "Slug", render: (p) => <Mono>{p.slug}</Mono> },
    { key: "status", header: "Status", render: (p) => <StatusPill value={p.status} /> },
    { key: "description", header: "Description", render: (p) => p.description || "—" },
    { key: "created", header: "Created", render: (p) => formatDate(p.created_at) },
  ];
  return (
    <>
      {manage && !open && (
        <div className="row-actions">
          <Action label="New project" onClick={() => setOpen(true)} />
        </div>
      )}
      {manage && open && (
        <NameForm
          submitLabel="Create project"
          pending={create.isPending}
          error={create.error}
          onSubmit={(values) => {
            const description = values.description.trim();
            create.mutate(
              description === "" ? { name: values.name } : { name: values.name, description },
              { onSuccess: () => setOpen(false) },
            );
          }}
        />
      )}
      <DataTable
        caption="Projects"
        columns={columns}
        rows={query.data ?? []}
        isLoading={query.isPending}
        isError={query.isError}
        error={query.error}
        empty="No projects yet."
      />
    </>
  );
}

export function TeamsBody() {
  const query = useTeams();
  const create = useCreateTeam();
  const manage = useManager();
  const [open, setOpen] = useState(false);
  const columns: readonly Column<Team>[] = [
    { key: "name", header: "Name", render: (t) => t.name },
    { key: "description", header: "Description", render: (t) => t.description || "—" },
    { key: "created", header: "Created", render: (t) => formatDate(t.created_at) },
  ];
  return (
    <>
      {manage && !open && (
        <div className="row-actions">
          <Action label="New team" onClick={() => setOpen(true)} />
        </div>
      )}
      {manage && open && (
        <NameForm
          submitLabel="Create team"
          pending={create.isPending}
          error={create.error}
          onSubmit={(values) => {
            const description = values.description.trim();
            create.mutate(
              description === "" ? { name: values.name } : { name: values.name, description },
              { onSuccess: () => setOpen(false) },
            );
          }}
        />
      )}
      <DataTable
        caption="Teams"
        columns={columns}
        rows={query.data ?? []}
        isLoading={query.isPending}
        isError={query.isError}
        error={query.error}
        empty="No teams yet."
      />
    </>
  );
}

export function ServiceAccountsBody() {
  const query = useServiceAccounts();
  const columns: readonly Column<ServiceAccount>[] = [
    { key: "name", header: "Name", render: (s) => s.name },
    { key: "description", header: "Description", render: (s) => s.description || "—" },
    { key: "status", header: "Status", render: (s) => <StatusPill value={s.status} /> },
    { key: "created", header: "Created", render: (s) => formatDate(s.created_at) },
  ];
  return (
    <DataTable
      caption="Service accounts"
      columns={columns}
      rows={query.data ?? []}
      isLoading={query.isPending}
      isError={query.isError}
      error={query.error}
      empty="No service accounts yet."
    />
  );
}

export function ApiCredentialsBody() {
  const query = useApiCredentials();
  const revoke = useRevokeApiCredential();
  const create = useCreateApiCredential();
  const manage = useManager();
  const [open, setOpen] = useState(false);
  const [revealed, setRevealed] = useState<string | null>(null);
  const form = useForm<ApiCredentialFormValues>({
    resolver: zodResolver(apiCredentialFormSchema),
    defaultValues: { name: "", scope: "gateway", service_account_id: "" },
  });

  useEffect(() => {
    if (create.isSuccess && create.data) {
      setRevealed(create.data.api_key);
      create.reset();
    }
  }, [create.isSuccess, create.data, create.reset]);

  const columns: readonly Column<ApiCredential>[] = [
    { key: "name", header: "Name", render: (c) => c.name },
    { key: "prefix", header: "Key prefix", render: (c) => <Mono>{c.key_prefix}</Mono> },
    { key: "scope", header: "Scope", render: (c) => c.scope },
    { key: "status", header: "Status", render: (c) => <StatusPill value={c.status} /> },
    { key: "used", header: "Last used", render: (c) => never(c.last_used_at, "Never") },
    ...(manage
      ? [
          {
            key: "actions",
            header: "",
            align: "right" as const,
            render: (c: ApiCredential) => (
              <Actions>
                <Action
                  label="Revoke"
                  tone="danger"
                  disabled={c.status === "revoked"}
                  pending={revoke.isPending}
                  onClick={() => revoke.mutate(c.id)}
                />
              </Actions>
            ),
          },
        ]
      : []),
  ];
  return (
    <>
      {revealed !== null && (
        <SecretReveal label="New API key" value={revealed} onDismiss={() => setRevealed(null)} />
      )}
      {manage && !open && (
        <div className="row-actions">
          <Action label="New API key" onClick={() => setOpen(true)} />
        </div>
      )}
      {manage && open && (
        <form
          className="inline-form"
          onSubmit={form.handleSubmit((values) => {
            const serviceAccountId = values.service_account_id.trim();
            create.mutate(
              serviceAccountId === ""
                ? { name: values.name, scope: values.scope }
                : { name: values.name, scope: values.scope, service_account_id: serviceAccountId },
              {
                onSuccess: () => {
                  form.reset();
                  setOpen(false);
                },
              },
            );
          })}
        >
          <div className="form-grid">
            <Field label="Name" error={form.formState.errors.name?.message}>
              <input className="input" {...form.register("name")} />
            </Field>
            <Field label="Scope">
              <select className="input" {...form.register("scope")}>
                {SCOPES.map((scope) => (
                  <option key={scope} value={scope}>
                    {scope}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Service account ID" error={form.formState.errors.service_account_id?.message}>
              <input
                className="input"
                autoComplete="off"
                placeholder="Optional"
                {...form.register("service_account_id")}
              />
            </Field>
          </div>
          <SubmitButton pending={create.isPending} label="Create key" pendingLabel="Creating…" />
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => setOpen(false)}
            disabled={create.isPending}
          >
            Cancel
          </button>
          {create.error !== null && <ErrorNote error={create.error} />}
        </form>
      )}
      <DataTable
        caption="API credentials"
        columns={columns}
        rows={query.data ?? []}
        isLoading={query.isPending}
        isError={query.isError}
        error={query.error}
        empty="No API credentials yet."
      />
      <EmptyNote>
        The full key is returned exactly once at creation and is never stored in plain text.
      </EmptyNote>
    </>
  );
}

export function GatewayCredentialsBody() {
  const query = useGatewayCredentials();
  const rotate = useRotateGatewayCredential();
  const revoke = useRevokeGatewayCredential();
  const expire = useExpireGatewayCredential();
  const restore = useRestoreGatewayCredential();
  const create = useCreateGatewayCredential();
  const members = useMembers();
  const serviceAccounts = useServiceAccounts();
  const manage = useManager();
  const [open, setOpen] = useState(false);
  const [revealed, setRevealed] = useState<string | null>(null);
  const form = useForm<GatewayCredentialFormValues>({
    resolver: zodResolver(gatewayCredentialFormSchema),
    defaultValues: {
      name: "",
      subject_type: "user",
      subject_id: "",
      scopes: ["gateway"],
      expires_at: "",
    },
  });
  const subjectType = form.watch("subject_type");

  useEffect(() => {
    if (create.isSuccess && create.data) {
      setRevealed(create.data.key_prefix);
      create.reset();
    }
  }, [create.isSuccess, create.data, create.reset]);

  useEffect(() => {
    if (rotate.isSuccess && rotate.data) {
      setRevealed(rotate.data.key_prefix);
      rotate.reset();
    }
  }, [rotate.isSuccess, rotate.data, rotate.reset]);

  const columns: readonly Column<GatewayCredential>[] = [
    { key: "name", header: "Name", render: (c) => c.name },
    {
      key: "subject",
      header: "Subject",
      render: (c) => (
        <Mono>
          {c.subject_type}/{c.subject_id.slice(0, 8)}
        </Mono>
      ),
    },
    { key: "scopes", header: "Scopes", render: (c) => c.scopes.join(", ") || "—" },
    { key: "status", header: "Status", render: (c) => <StatusPill value={c.status} /> },
    { key: "used", header: "Last used", render: (c) => never(c.last_used_at, "Never") },
    { key: "expires", header: "Expires", render: (c) => never(c.expires_at, "No expiry") },
    ...(manage
      ? [
          {
            key: "actions",
            header: "",
            align: "right" as const,
            render: (c: GatewayCredential) => (
              <Actions>
                <Action
                  label="Rotate"
                  pending={rotate.isPending}
                  disabled={c.status === "revoked"}
                  onClick={() => rotate.mutate({ id: c.id })}
                />
                {c.status === "expired" ? (
                  <Action
                    label="Restore"
                    pending={restore.isPending}
                    onClick={() => restore.mutate(c.id)}
                  />
                ) : (
                  <Action
                    label="Expire"
                    pending={expire.isPending}
                    disabled={c.status === "revoked"}
                    onClick={() => expire.mutate(c.id)}
                  />
                )}
                <Action
                  label="Revoke"
                  tone="danger"
                  pending={revoke.isPending}
                  disabled={c.status === "revoked"}
                  onClick={() => revoke.mutate(c.id)}
                />
              </Actions>
            ),
          },
        ]
      : []),
  ];

  return (
    <>
      {revealed !== null && (
        <SecretReveal
          label="New gateway key"
          value={revealed}
          onDismiss={() => {
            setRevealed(null);
            rotate.reset();
          }}
        />
      )}
      {manage && !open && (
        <div className="row-actions">
          <Action label="New gateway key" onClick={() => setOpen(true)} />
        </div>
      )}
      {manage && open && (
        <form
          className="inline-form"
          onSubmit={form.handleSubmit((values) => {
            const expiresAt = values.expires_at.trim();
            const base = {
              name: values.name,
              subject_type: values.subject_type,
              subject_id: values.subject_id,
              scopes: values.scopes,
            };
            create.mutate(
              expiresAt === ""
                ? base
                : { ...base, expires_at: new Date(expiresAt).toISOString() },
              {
                onSuccess: () => {
                  form.reset();
                  setOpen(false);
                },
              },
            );
          })}
        >
          <div className="form-grid">
            <Field label="Name" error={form.formState.errors.name?.message}>
              <input className="input" {...form.register("name")} />
            </Field>
            <Field label="Subject type">
              <select
                className="input"
                {...form.register("subject_type", {
                  onChange: (event) => {
                    form.setValue("subject_id", "");
                    void event.target.value;
                  },
                })}
              >
                {SUBJECT_TYPES.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Subject" error={form.formState.errors.subject_id?.message}>
              <select className="input" {...form.register("subject_id")}>
                <option value="">Select…</option>
                {subjectType === "user"
                  ? (members.data ?? []).map((member) => (
                      <option key={member.user.id} value={member.user.id}>
                        {member.user.display_name || member.user.email} ({member.user.email})
                      </option>
                    ))
                  : (serviceAccounts.data ?? []).map((account) => (
                      <option key={account.id} value={account.id}>
                        {account.name}
                      </option>
                    ))}
              </select>
            </Field>
            <Field label="Expires" error={form.formState.errors.expires_at?.message}>
              <input className="input" type="datetime-local" {...form.register("expires_at")} />
            </Field>
          </div>
          <fieldset className="field">
            <legend>Scopes</legend>
            <div className="form-grid">
              {SCOPES.map((scope) => (
                <label key={scope} className="field">
                  <span>
                    <input type="checkbox" value={scope} {...form.register("scopes")} /> {scope}
                  </span>
                </label>
              ))}
            </div>
            {form.formState.errors.scopes?.message !== undefined && (
              <span className="field-error">{form.formState.errors.scopes.message}</span>
            )}
          </fieldset>
          <SubmitButton pending={create.isPending} label="Create key" pendingLabel="Creating…" />
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => setOpen(false)}
            disabled={create.isPending}
          >
            Cancel
          </button>
          {create.error !== null && <ErrorNote error={create.error} />}
        </form>
      )}
      <DataTable
        caption="Gateway credentials"
        columns={columns}
        rows={query.data ?? []}
        isLoading={query.isPending}
        isError={query.isError}
        error={query.error}
        empty="No gateway credentials yet."
      />
    </>
  );
}

export function ProviderAccountsBody() {
  const query = useProviderAccounts();
  const create = useCreateProviderAccount();
  const manage = useManager();
  const [open, setOpen] = useState(false);
  const form = useForm<ProviderAccountFormValues>({
    resolver: zodResolver(providerAccountFormSchema),
    defaultValues: { provider_kind: "openai", name: "", base_url: "" },
  });
  const providerKind = form.watch("provider_kind");
  const columns: readonly Column<ProviderAccount>[] = [
    { key: "name", header: "Name", render: (a) => a.name },
    { key: "kind", header: "Provider", render: (a) => <StatusPill value={a.provider_kind} /> },
    { key: "base", header: "Base URL", render: (a) => (a.base_url ? <Mono>{a.base_url}</Mono> : "—") },
    { key: "status", header: "Status", render: (a) => <StatusPill value={a.status} /> },
    { key: "created", header: "Created", render: (a) => formatDate(a.created_at) },
  ];
  return (
    <>
      {manage && !open && (
        <div className="row-actions">
          <Action label="New provider account" onClick={() => setOpen(true)} />
        </div>
      )}
      {manage && open && (
        <form
          className="inline-form"
          onSubmit={form.handleSubmit((values) => {
            const baseUrl = values.base_url.trim();
            create.mutate(
              baseUrl === ""
                ? { provider_kind: values.provider_kind, name: values.name }
                : { provider_kind: values.provider_kind, name: values.name, base_url: baseUrl },
              {
                onSuccess: () => {
                  form.reset();
                  setOpen(false);
                },
              },
            );
          })}
        >
          <div className="form-grid">
            <Field label="Provider">
              <select className="input" {...form.register("provider_kind")}>
                {PROVIDER_KINDS.map((kind) => (
                  <option key={kind} value={kind}>
                    {kind}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Name" error={form.formState.errors.name?.message}>
              <input className="input" {...form.register("name")} />
            </Field>
            <Field
              label={providerKind === "custom" ? "Base URL" : "Base URL (optional)"}
              error={form.formState.errors.base_url?.message}
            >
              <input
                className="input"
                inputMode="url"
                autoComplete="off"
                placeholder={providerKind === "custom" ? "https://…" : ""}
                required={providerKind === "custom"}
                {...form.register("base_url")}
              />
            </Field>
          </div>
          <SubmitButton pending={create.isPending} label="Create account" pendingLabel="Creating…" />
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => setOpen(false)}
            disabled={create.isPending}
          >
            Cancel
          </button>
          {create.error !== null && <ErrorNote error={create.error} />}
        </form>
      )}
      <DataTable
        caption="Provider accounts"
        columns={columns}
        rows={query.data ?? []}
        isLoading={query.isPending}
        isError={query.isError}
        error={query.error}
        empty="No provider accounts yet."
      />
    </>
  );
}

export function ProviderCredentialsBody() {
  const query = useProviderCredentials();
  const accounts = useProviderAccounts();
  const create = useCreateProviderCredential();
  const rotate = useRotateProviderCredential();
  const revoke = useRevokeProviderCredential();
  const manage = useManager();
  const [rotatingId, setRotatingId] = useState<string | null>(null);
  const [secret, setSecret] = useState("");
  const [open, setOpen] = useState(false);
  const form = useForm<ProviderCredentialFormValues>({
    resolver: zodResolver(providerCredentialFormSchema),
    defaultValues: { accountId: "", name: "", kind: "api_key", secret: "" },
  });

  function submitRotation(event: FormEvent) {
    event.preventDefault();
    if (rotatingId !== null && secret !== "") {
      rotate.mutate(
        { id: rotatingId, secret },
        { onSuccess: () => setSecret("") },
      );
      setRotatingId(null);
    }
  }
  const columns: readonly Column<ProviderCredential>[] = [
    { key: "name", header: "Name", render: (c) => c.name },
    {
      key: "account",
      header: "Account",
      render: (c) => <Mono>{c.provider_account_id.slice(0, 8)}</Mono>,
    },
    { key: "kind", header: "Kind", render: (c) => <StatusPill value={c.kind} /> },
    {
      key: "secret",
      header: "Secret",
      render: (c) => (
        <Mono>
          {c.fingerprint}…{c.tail}
        </Mono>
      ),
    },
    { key: "status", header: "Status", render: (c) => <StatusPill value={c.status} /> },
    ...(manage
      ? [
          {
            key: "actions",
            header: "",
            align: "right" as const,
            render: (c: ProviderCredential) => (
              <Actions>
                <Action
                  label="Rotate"
                  pending={rotate.isPending}
                  disabled={c.status === "revoked"}
                  onClick={() => {
                    setSecret("");
                    setRotatingId(c.id);
                  }}
                />
                <Action
                  label="Revoke"
                  tone="danger"
                  pending={revoke.isPending}
                  disabled={c.status === "revoked"}
                  onClick={() => revoke.mutate(c.id)}
                />
              </Actions>
            ),
          },
        ]
      : []),
  ];
  return (
    <>
      {manage && !open && (
        <div className="row-actions">
          <Action label="New provider credential" onClick={() => setOpen(true)} />
        </div>
      )}
      {manage && open && (
        <form
          className="inline-form"
          onSubmit={form.handleSubmit((values) => {
            create.mutate(
              {
                accountId: values.accountId,
                name: values.name,
                kind: values.kind,
                secret: values.secret,
              },
              {
                onSuccess: () => {
                  form.reset();
                  setOpen(false);
                },
              },
            );
          })}
        >
          <div className="form-grid">
            <Field label="Account" error={form.formState.errors.accountId?.message}>
              <select className="input" {...form.register("accountId")}>
                <option value="">Select…</option>
                {(accounts.data ?? []).map((account) => (
                  <option key={account.id} value={account.id}>
                    {account.name} ({account.provider_kind})
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Name" error={form.formState.errors.name?.message}>
              <input className="input" {...form.register("name")} />
            </Field>
            <Field label="Kind">
              <select className="input" {...form.register("kind")}>
                {CREDENTIAL_KINDS.map((kind) => (
                  <option key={kind} value={kind}>
                    {kind}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Secret" error={form.formState.errors.secret?.message}>
              <input
                className="input"
                type="password"
                autoComplete="off"
                {...form.register("secret")}
              />
            </Field>
          </div>
          <SubmitButton
            pending={create.isPending}
            label="Create credential"
            pendingLabel="Creating…"
          />
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => setOpen(false)}
            disabled={create.isPending}
          >
            Cancel
          </button>
          {create.error !== null && <ErrorNote error={create.error} />}
          <EmptyNote>The secret is sent once and is never returned by the API.</EmptyNote>
        </form>
      )}
      {rotatingId !== null && (
        <form className="inline-form" onSubmit={submitRotation}>
          <label className="field">
            <span>New secret</span>
            <input
              className="input"
              type="password"
              autoComplete="off"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              required
            />
          </label>
          <button type="submit" className="btn" disabled={secret === "" || rotate.isPending}>
            {rotate.isPending ? "Replacing…" : "Replace secret"}
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => setRotatingId(null)}
            disabled={rotate.isPending}
          >
            Cancel
          </button>
        </form>
      )}
      <DataTable
        caption="Provider credentials"
        columns={columns}
        rows={query.data ?? []}
        isLoading={query.isPending}
        isError={query.isError}
        error={query.error}
        empty="No provider credentials yet."
      />
      <EmptyNote>
        Stored secrets are never readable. Identification is limited to a fingerprint and the
        last four characters.
      </EmptyNote>
    </>
  );
}

export function ModelsBody() {
  const query = useModels();
  const sync = useSyncModels();
  const manage = useManager();
  const [open, setOpen] = useState(false);
  const form = useForm<SyncModelsValues>({
    resolver: zodResolver(syncModelsSchema),
    defaultValues: { providerKind: "openai", apiKey: "" },
  });
  const columns: readonly Column<Model>[] = [
    { key: "name", header: "Model", render: (m) => m.display_name || m.model_slug },
    { key: "slug", header: "Slug", render: (m) => <Mono>{m.model_slug}</Mono> },
    { key: "provider", header: "Provider", render: (m) => <StatusPill value={m.provider_kind} /> },
    {
      key: "active",
      header: "State",
      render: (m) => <StatusPill value={m.is_active ? "active" : "inactive"} />,
    },
    { key: "updated", header: "Updated", render: (m) => formatDate(m.updated_at) },
  ];
  return (
    <>
      {manage && !open && (
        <div className="row-actions">
          <Action label="Sync models" onClick={() => setOpen(true)} />
        </div>
      )}
      {manage && open && (
        <form
          className="inline-form"
          onSubmit={form.handleSubmit((values) =>
            sync.mutate(
              { providerKind: values.providerKind, apiKey: values.apiKey },
              {
                onSuccess: () => {
                  form.reset();
                  setOpen(false);
                },
              },
            ),
          )}
        >
          <div className="form-grid">
            <Field label="Provider">
              <select className="input" {...form.register("providerKind")}>
                {PROVIDER_KINDS.map((kind) => (
                  <option key={kind} value={kind}>
                    {kind}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="API key" error={form.formState.errors.apiKey?.message}>
              <input
                className="input"
                type="password"
                autoComplete="off"
                {...form.register("apiKey")}
              />
            </Field>
          </div>
          <SubmitButton pending={sync.isPending} label="Sync" pendingLabel="Syncing…" />
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => setOpen(false)}
            disabled={sync.isPending}
          >
            Cancel
          </button>
          {sync.error !== null && <ErrorNote error={sync.error} />}
          <EmptyNote>
            The key is used to fetch the catalog and is not stored. Manage it under Provider
            credentials instead.
          </EmptyNote>
        </form>
      )}
      <DataTable
        caption="Model catalog"
        columns={columns}
        rows={query.data ?? []}
        isLoading={query.isPending}
        isError={query.isError}
        error={query.error}
        empty="No models synced yet."
      />
    </>
  );
}

export function ProviderHealthBody() {
  const query = useProviderHealth();
  const columns: readonly Column<ProviderHealth>[] = [
    { key: "provider", header: "Provider", render: (h) => <StatusPill value={h.provider_kind} /> },
    { key: "status", header: "Status", render: (h) => <StatusPill value={h.status} /> },
    {
      key: "detail",
      header: "Detail",
      render: (h) => h.status_reason || h.last_error || "—",
    },
    { key: "checked", header: "Checked", render: (h) => formatDateTime(h.checked_at) },
  ];
  return (
    <DataTable
      caption="Provider health"
      columns={columns}
      rows={query.data ?? []}
      isLoading={query.isPending}
      isError={query.isError}
      error={query.error}
      empty="No provider health checks recorded yet."
    />
  );
}
