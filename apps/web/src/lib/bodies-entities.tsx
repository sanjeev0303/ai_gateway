"use client";

import { useMemo, useState } from "react";

import {
  DataTable,
  Mono,
  StatusPill,
  formatDate,
  never,
  type Column,
} from "@/components/ui";
import {
  useMembers,
  useModels,
  useProjects,
  useTeams,
} from "@/lib/queries";
import type {
  Member,
  Model,
  Project,
  Team,
} from "@/lib/control-api";
import { DetailTile, PendingColumns } from "@/lib/spec";

const USER_DETAIL_AREAS = [
  "usage",
  "cost",
  "models",
  "providers",
  "projects",
  "conversations",
  "credentials",
  "policies",
  "audit",
] as const;

function UserDetail({ member }: { readonly member: Member }) {
  return (
    <div className="dash-panel">
      <p className="dash-panel-title">{member.user.display_name || member.user.email}</p>
      <div className="card-grid">
        <DetailTile label="Email" value={member.user.email} />
        <DetailTile label="Role" value={member.role} />
        <DetailTile label="Status" value={member.user.status} />
        <DetailTile label="Member since" value={formatDate(member.member_since)} />
        <DetailTile label="Last activity" value={never(member.user.last_login_at, "Never")} />
        <DetailTile label="User ID" value={member.user.id} mono />
      </div>
      <p className="dash-note">
        Remaining detail areas — {USER_DETAIL_AREAS.join(", ")} — need a per-user aggregate endpoint
        and are not shown. Team membership needs an organization-wide membership rollup.
      </p>
    </div>
  );
}

export function UsersBody() {
  const members = useMembers();
  const [selected, setSelected] = useState<string | null>(null);
  const current = useMemo(
    () => members.data?.find((m) => m.user.id === selected) ?? null,
    [members.data, selected],
  );
  const columns: readonly Column<Member>[] = [
    { key: "user", header: "User", render: (m) => m.user.display_name || m.user.email },
    { key: "email", header: "Email", render: (m) => <Mono>{m.user.email}</Mono> },
    { key: "role", header: "Role", render: (m) => <StatusPill value={m.role} /> },
    { key: "status", header: "Status", render: (m) => <StatusPill value={m.user.status} /> },
    { key: "login", header: "Last activity", render: (m) => never(m.user.last_login_at, "Never") },
    {
      key: "detail",
      header: "",
      align: "right",
      render: (m) => (
        <button
          type="button"
          className="btn btn-ghost btn-sm"
          onClick={() => setSelected(selected === m.user.id ? null : m.user.id)}
        >
          {selected === m.user.id ? "Close" : "Detail"}
        </button>
      ),
    },
  ];
  return (
    <>
      <DataTable
        caption="Users by role and last activity"
        columns={columns}
        rows={members.data ?? []}
        isLoading={members.isPending}
        isError={members.isError}
        error={members.error}
        empty="No members yet."
      />
      {current !== null && <UserDetail member={current} />}
      <PendingColumns
        subject="Per-user"
        pending={["Team", "Requests", "Tokens", "Spend", "Current model"]}
        needs={["GET /v1/usage/aggregates?user_id", "GET /v1/members/usage"]}
      />
    </>
  );
}

export function TeamsObserveBody() {
  const teams = useTeams();
  const columns: readonly Column<Team>[] = [
    { key: "name", header: "Team", render: (t) => t.name },
    { key: "description", header: "Description", render: (t) => t.description || "—" },
    { key: "created", header: "Created", render: (t) => formatDate(t.created_at) },
  ];
  return (
    <>
      <DataTable
        caption="Teams"
        columns={columns}
        rows={teams.data ?? []}
        isLoading={teams.isPending}
        isError={teams.isError}
        error={teams.error}
        empty="No teams yet."
      />
      <PendingColumns
        subject="Per-team"
        pending={["Members", "Requests", "Tokens", "Spend"]}
        needs={["GET /v1/usage/aggregates?team_id"]}
      />
    </>
  );
}

export function ProjectsObserveBody() {
  const projects = useProjects();
  const columns: readonly Column<Project>[] = [
    { key: "name", header: "Project", render: (p) => p.name },
    { key: "slug", header: "Slug", render: (p) => <Mono>{p.slug}</Mono> },
    { key: "status", header: "Status", render: (p) => <StatusPill value={p.status} /> },
    { key: "created", header: "Created", render: (p) => formatDate(p.created_at) },
  ];
  return (
    <>
      <DataTable
        caption="Projects"
        columns={columns}
        rows={projects.data ?? []}
        isLoading={projects.isPending}
        isError={projects.isError}
        error={projects.error}
        empty="No projects yet."
      />
      <PendingColumns
        subject="Per-project"
        pending={[
          "Requests",
          "Tokens",
          "Spend",
          "Users",
          "Providers",
          "Models",
          "Recent activity",
          "Budget",
          "Policy",
        ]}
        needs={["GET /v1/usage/aggregates?project_id", "GET /v1/budgets"]}
      />
    </>
  );
}

export function ModelsObserveBody() {
  const models = useModels();
  const columns: readonly Column<Model>[] = [
    { key: "provider", header: "Provider", render: (m) => <StatusPill value={m.provider_kind} /> },
    { key: "model", header: "Model", render: (m) => m.display_name || m.model_slug },
    { key: "slug", header: "Model ID", render: (m) => <Mono>{m.model_slug}</Mono> },
    { key: "status", header: "Status", render: (m) => <StatusPill value={m.is_active ? "active" : "inactive"} /> },
    { key: "updated", header: "Last synced", render: (m) => formatDate(m.updated_at) },
  ];
  return (
    <>
      <DataTable
        caption="Models by provider and status"
        columns={columns}
        rows={models.data ?? []}
        isLoading={models.isPending}
        isError={models.isError}
        error={models.error}
        empty="No models synced yet."
      />
      <PendingColumns
        subject="Per-model"
        pending={[
          "Capabilities",
          "Requests",
          "Tokens",
          "Cost",
          "Error rate",
          "P95 latency",
          "Access policy",
        ]}
        needs={["GET /v1/usage/aggregates?model_id", "GET /v1/requests/aggregates", "GET /v1/policies"]}
      />
    </>
  );
}
