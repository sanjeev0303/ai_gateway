import { cache } from "react";

import { getApiClient } from "@/lib/session";

export interface DirectoryEntry {
  id: string;
  name: string;
  /** Second line in pickers, such as an email address. */
  detail?: string;
}

export interface Directory {
  teams: DirectoryEntry[];
  projects: DirectoryEntry[];
  users: DirectoryEntry[];
  serviceAccounts: DirectoryEntry[];
  policies: DirectoryEntry[];
  credentials: DirectoryEntry[];
  models: DirectoryEntry[];
}

const byName = (a: DirectoryEntry, b: DirectoryEntry) => a.name.localeCompare(b.name);

/**
 * The names behind the ids the API returns, so screens show "Finance" rather
 * than a UUID. A failed list leaves that kind empty: callers fall back to a
 * short id rather than failing the page.
 */
export const getDirectory = cache(async (): Promise<Directory> => {
  const client = await getApiClient();
  const [teams, projects, members, serviceAccounts, policies, models, credentials] = await Promise.all([
    client.teams.list().catch(() => ({ items: [] })),
    client.projects.list().catch(() => ({ items: [] })),
    client.members.list().catch(() => ({ items: [] })),
    client.serviceAccounts.list().catch(() => ({ items: [] })),
    client.policies.list().catch(() => ({ items: [] })),
    client.models.list().catch(() => ({ items: [] })),
    client.credentials.listGateway().catch(() => ({ items: [] })),
  ]);

  return {
    teams: teams.items.map((t) => ({ id: t.id, name: t.name })).sort(byName),
    projects: projects.items.map((p) => ({ id: p.id, name: p.name })).sort(byName),
    users: members.items
      .map((m) => ({ id: m.user.id, name: m.user.display_name || m.user.email, detail: m.user.email }))
      .sort(byName),
    serviceAccounts: serviceAccounts.items.map((s) => ({ id: s.id, name: s.name })).sort(byName),
    policies: policies.items.map((p) => ({ id: p.id ?? "", name: p.name ?? "" })).sort(byName),
    credentials: credentials.items
      .map((c) => ({ id: c.id, name: c.name, detail: c.key_prefix }))
      .sort(byName),
    models: models.items
      .map((m) => ({ id: m.model_slug, name: m.model_slug, detail: m.provider_kind }))
      .sort(byName),
  };
});

export function nameMap(entries: readonly DirectoryEntry[]): Record<string, string> {
  const names: Record<string, string> = {};
  for (const entry of entries) names[entry.id] = entry.name;
  return names;
}

/** A name for an id, or a short form of the id when the name is unknown. */
export function nameFor(names: Record<string, string>, id?: string | null): string {
  if (!id) return "—";
  return names[id] ?? `${id.slice(0, 8)}…`;
}
