import { z } from "zod";

const CONTROL_API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

export const ROLES = [
  "owner",
  "admin",
  "auditor",
  "billing_admin",
  "member",
] as const;
export type Role = (typeof ROLES)[number];

export function isRole(value: string): value is Role {
  return (ROLES as readonly string[]).includes(value);
}

export function canManage(role: string): boolean {
  return role === "owner" || role === "admin";
}

export const READ_PERMISSIONS = [
  "members.read",
  "teams.read",
  "projects.read",
  "providers.read",
  "credentials.read",
  "usage.read",
  "policies.read",
  "audit.read",
  "alerts.read",
  "billing.read",
  "settings.read",
] as const;
export type ReadPermission = (typeof READ_PERMISSIONS)[number];

const READS_BY_ROLE: Record<Role, readonly ReadPermission[]> = {
  owner: READ_PERMISSIONS,
  admin: READ_PERMISSIONS,
  auditor: [
    "members.read",
    "teams.read",
    "projects.read",
    "providers.read",
    "credentials.read",
    "usage.read",
    "policies.read",
    "audit.read",
    "alerts.read",
    "settings.read",
  ],
  billing_admin: ["members.read", "teams.read", "projects.read", "usage.read", "billing.read"],
  member: ["members.read", "teams.read", "projects.read", "usage.read"],
};

export function canRead(role: string, permission: ReadPermission): boolean {
  const reads = isRole(role) ? READS_BY_ROLE[role] : undefined;
  return reads !== undefined && reads.includes(permission);
}

const claimsSchema = z.object({
  subject: z.string(),
  email: z.string(),
  display_name: z.string(),
  organization_id: z.string(),
  role: z.string(),
  kind: z.string(),
  issued_at: z.string(),
  expires_at: z.string(),
});

const sessionSchema = z.object({
  claims: claimsSchema,
  csrf_token: z.string().optional(),
});

export type SessionClaims = z.infer<typeof claimsSchema>;
export type Session = z.infer<typeof sessionSchema>;

const userSchema = z.object({
  id: z.string(),
  email: z.string(),
  display_name: z.string(),
  status: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
  last_login_at: z.string().optional(),
});

const memberSchema = z.object({
  user: userSchema,
  role: z.string(),
  member_since: z.string(),
});

const projectSchema = z.object({
  id: z.string(),
  organization_id: z.string(),
  name: z.string(),
  slug: z.string(),
  description: z.string(),
  status: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

const teamSchema = z.object({
  id: z.string(),
  organization_id: z.string(),
  name: z.string(),
  description: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

const serviceAccountSchema = z.object({
  id: z.string(),
  organization_id: z.string(),
  name: z.string(),
  description: z.string(),
  status: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

const apiCredentialSchema = z.object({
  id: z.string(),
  organization_id: z.string(),
  name: z.string(),
  key_prefix: z.string(),
  scope: z.string(),
  status: z.string(),
  service_account_id: z.string().optional(),
  last_used_at: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

const gatewayCredentialSchema = z.object({
  id: z.string(),
  organization_id: z.string(),
  name: z.string(),
  key_prefix: z.string(),
  subject_type: z.string(),
  subject_id: z.string(),
  team_id: z.string().optional(),
  project_id: z.string().optional(),
  scopes: z.array(z.string()),
  status: z.string(),
  expires_at: z.string().optional(),
  last_used_at: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

const providerAccountSchema = z.object({
  id: z.string(),
  organization_id: z.string(),
  provider_kind: z.string(),
  name: z.string(),
  base_url: z.string().optional(),
  status: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

const providerCredentialSchema = z.object({
  id: z.string(),
  organization_id: z.string(),
  provider_account_id: z.string(),
  name: z.string(),
  kind: z.string(),
  fingerprint: z.string(),
  tail: z.string(),
  status: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

const modelSchema = z.object({
  id: z.string(),
  provider_kind: z.string(),
  model_slug: z.string(),
  display_name: z.string(),
  is_active: z.boolean(),
  created_at: z.string(),
  updated_at: z.string(),
});

const providerHealthSchema = z.object({
  provider_kind: z.string(),
  status: z.string(),
  status_reason: z.string().optional(),
  last_error: z.string().optional(),
  checked_at: z.string(),
});

const activityEventSchema = z.object({
  request_id: z.string(),
  organization_id: z.string(),
  user_id: z.string().optional(),
  user_name: z.string().optional(),
  service_account: z.string().optional(),
  project_id: z.string().optional(),
  project_name: z.string().optional(),
  team_id: z.string().optional(),
  provider: z.string(),
  model: z.string(),
  status: z.string(),
  started_at: z.string(),
  current_state: z.enum(["active", "idle", "completed", "failed"]),
});

const invitationSchema = z.object({
  id: z.string(),
  organization_id: z.string(),
  email: z.string(),
  role: z.string(),
  status: z.string(),
  expires_at: z.string(),
  invited_by: z.string().optional(),
  accepted_by: z.string().optional(),
  accepted_at: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

const teamMemberSchema = z.object({
  team_id: z.string(),
  user_id: z.string(),
  role: z.string(),
  member_since: z.string(),
});

const projectMemberSchema = z.object({
  project_id: z.string(),
  user_id: z.string(),
  role: z.string(),
  member_since: z.string(),
});

export type User = z.infer<typeof userSchema>;
export type Member = z.infer<typeof memberSchema>;
export type Project = z.infer<typeof projectSchema>;
export type Team = z.infer<typeof teamSchema>;
export type ServiceAccount = z.infer<typeof serviceAccountSchema>;
export type ApiCredential = z.infer<typeof apiCredentialSchema>;
export type GatewayCredential = z.infer<typeof gatewayCredentialSchema>;
export type ProviderAccount = z.infer<typeof providerAccountSchema>;
export type ProviderCredential = z.infer<typeof providerCredentialSchema>;
export type Model = z.infer<typeof modelSchema>;
export type ProviderHealth = z.infer<typeof providerHealthSchema>;
export type Invitation = z.infer<typeof invitationSchema>;
export type TeamMember = z.infer<typeof teamMemberSchema>;
export type ProjectMember = z.infer<typeof projectMemberSchema>;
export type ActivityEvent = z.infer<typeof activityEventSchema>;

function items<T extends z.ZodType>(schema: T) {
  return z.object({ items: z.array(schema) });
}

const membersList = items(memberSchema);
const projectsList = items(projectSchema);
const teamsList = items(teamSchema);
const serviceAccountsList = items(serviceAccountSchema);
const apiCredentialsList = items(apiCredentialSchema);
const gatewayCredentialsList = items(gatewayCredentialSchema);
const providerAccountsList = items(providerAccountSchema);
const providerCredentialsList = items(providerCredentialSchema);
const modelsList = items(modelSchema);
const providerHealthList = items(providerHealthSchema);
const invitationsList = items(invitationSchema);
const teamMembersList = items(teamMemberSchema);
const projectMembersList = items(projectMemberSchema);
const activityList = z.object({ data: z.array(activityEventSchema) });

const createApiCredentialSchema = z.object({
  credential: apiCredentialSchema,
  api_key: z.string(),
});

export type CreatedApiCredential = z.infer<typeof createApiCredentialSchema>;

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

async function toApiError(response: Response): Promise<ApiError> {
  let code = "request_failed";
  let message = `Request failed with status ${response.status}`;
  if (response.headers.get("Content-Type")?.includes("application/json")) {
    const body = (await response.json()) as { error?: { code?: string; message?: string } };
    if (body.error) {
      code = body.error.code ?? code;
      message = body.error.message ?? message;
    }
  }
  return new ApiError(response.status, code, message);
}

async function send(path: string, init: RequestInit, csrfToken?: string): Promise<Response> {
  const response = await fetch(`${CONTROL_API}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      ...(init.body === undefined ? {} : { "Content-Type": "application/json" }),
      ...(csrfToken === undefined ? {} : { "X-CSRF-Token": csrfToken }),
      ...init.headers,
    },
  });
  if (!response.ok) {
    throw await toApiError(response);
  }
  return response;
}

async function request<T>(path: string, init: RequestInit = {}, csrfToken?: string): Promise<T> {
  const response = await send(path, init, csrfToken);
  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

async function get<T>(path: string, schema: z.ZodType<T>): Promise<T> {
  return schema.parse(await request<unknown>(path));
}

function post<T>(path: string, body: unknown, schema: z.ZodType<T>, csrfToken?: string): Promise<T> {
  return send(path, { method: "POST", body: JSON.stringify(body) }, csrfToken)
    .then((r) => r.json() as Promise<unknown>)
    .then((raw) => schema.parse(raw));
}

function del<T>(path: string, schema: z.ZodType<T>, csrfToken?: string): Promise<T> {
  return send(path, { method: "DELETE" }, csrfToken).then((r) =>
    r.status === 204
      ? (undefined as T)
      : (r.json() as Promise<unknown>).then((raw) => schema.parse(raw)),
  );
}

function patch<T>(path: string, body: unknown, schema: z.ZodType<T>, csrfToken?: string): Promise<T> {
  return send(
    path,
    { method: "PATCH", body: JSON.stringify(body) },
    csrfToken,
  )
    .then((r) => r.json() as Promise<unknown>)
    .then((raw) => schema.parse(raw));
}

const voidSchema = z.unknown();

export const signIn = (email: string, password: string): Promise<Session> =>
  post("/v1/auth/signin", { email, password }, sessionSchema);

export const signUp = (input: {
  readonly email: string;
  readonly password: string;
  readonly organization_name: string;
  readonly display_name?: string;
}): Promise<Session> => post("/v1/auth/signup", input, sessionSchema);

export const getSession = (): Promise<Session> => get("/v1/auth/session", sessionSchema);

export const refreshSession = (csrfToken: string): Promise<Session> =>
  post("/v1/auth/refresh", {}, sessionSchema, csrfToken);

export const signOut = (csrfToken: string): Promise<void> =>
  request("/v1/auth/logout", { method: "POST" }, csrfToken);

export const listMembers = (): Promise<{ items: Member[] }> =>
  get("/v1/members", membersList);

export const setMemberRole = (userId: string, role: string): Promise<unknown> =>
  patch(`/v1/members/${userId}/role`, { role }, voidSchema);

export const removeMember = (userId: string): Promise<unknown> =>
  del(`/v1/members/${userId}`, voidSchema);

export const listInvitations = (): Promise<{ items: Invitation[] }> =>
  get("/v1/invitations", invitationsList);

export const invite = (
  email: string,
  role: string,
  csrfToken: string,
): Promise<unknown> => post("/v1/invitations", { email, role }, voidSchema, csrfToken);

export const revokeInvitation = (id: string, csrfToken: string): Promise<unknown> =>
  del(`/v1/invitations/${id}`, voidSchema, csrfToken);

export const listProjects = (): Promise<{ items: Project[] }> =>
  get("/v1/projects", projectsList);

export const createProject = (
  input: { name: string; description?: string },
  csrfToken: string,
): Promise<Project> => post("/v1/projects", input, projectSchema, csrfToken);

export const listProjectMembers = (id: string): Promise<{ items: ProjectMember[] }> =>
  get(`/v1/projects/${id}/members`, projectMembersList);

export const listTeams = (): Promise<{ items: Team[] }> => get("/v1/teams", teamsList);

export const createTeam = (
  input: { name: string; description?: string },
  csrfToken: string,
): Promise<Team> => post("/v1/teams", input, teamSchema, csrfToken);

export const listTeamMembers = (id: string): Promise<{ items: TeamMember[] }> =>
  get(`/v1/teams/${id}/members`, teamMembersList);

export const removeTeamMember = (
  teamId: string,
  userId: string,
  csrfToken: string,
): Promise<unknown> => del(`/v1/teams/${teamId}/members/${userId}`, voidSchema, csrfToken);

export const listServiceAccounts = (): Promise<{ items: ServiceAccount[] }> =>
  get("/v1/service-accounts", serviceAccountsList);

export const listApiCredentials = (): Promise<{ items: ApiCredential[] }> =>
  get("/v1/credentials", apiCredentialsList);

export const createApiCredential = (
  input: { name: string; scope?: string; service_account_id?: string },
  csrfToken: string,
): Promise<CreatedApiCredential> =>
  post("/v1/credentials", input, createApiCredentialSchema, csrfToken);

export const revokeApiCredential = (id: string, csrfToken: string): Promise<unknown> =>
  del(`/v1/credentials/${id}`, voidSchema, csrfToken);

export const listGatewayCredentials = (): Promise<{ items: GatewayCredential[] }> =>
  get("/v1/gateway-credentials", gatewayCredentialsList);

export const createGatewayCredential = (
  input: {
    name: string;
    subject_type: string;
    subject_id: string;
    team_id?: string;
    project_id?: string;
    scopes: string[];
    expires_at?: string;
  },
  csrfToken: string,
): Promise<GatewayCredential> =>
  post("/v1/gateway-credentials", input, gatewayCredentialSchema, csrfToken);

export const revokeGatewayCredential = (id: string, csrfToken: string): Promise<unknown> =>
  del(`/v1/gateway-credentials/${id}`, voidSchema, csrfToken);

export const rotateGatewayCredential = (
  id: string,
  input: { name?: string; scopes?: string[]; expires_at?: string },
  csrfToken: string,
): Promise<GatewayCredential> =>
  post(`/v1/gateway-credentials/${id}/rotate`, input, gatewayCredentialSchema, csrfToken);

export const expireGatewayCredential = (id: string, csrfToken: string): Promise<unknown> =>
  post(`/v1/gateway-credentials/${id}/expire`, {}, voidSchema, csrfToken);

export const restoreGatewayCredential = (id: string, csrfToken: string): Promise<unknown> =>
  post(`/v1/gateway-credentials/${id}/restore`, {}, voidSchema, csrfToken);

export const listProviderAccounts = (): Promise<{ items: ProviderAccount[] }> =>
  get("/v1/provider-accounts", providerAccountsList);

export const createProviderAccount = (
  input: { provider_kind: string; name: string; base_url?: string },
  csrfToken: string,
): Promise<ProviderAccount> =>
  post("/v1/provider-accounts", input, providerAccountSchema, csrfToken);

export const listProviderCredentials = (): Promise<{ items: ProviderCredential[] }> =>
  get("/v1/provider-credentials", providerCredentialsList);

export const createProviderCredential = (
  accountId: string,
  input: { name: string; kind: string; secret: string },
  csrfToken: string,
): Promise<ProviderCredential> =>
  post(
    `/v1/provider-accounts/${accountId}/credentials`,
    input,
    providerCredentialSchema,
    csrfToken,
  );

export const revokeProviderCredential = (id: string, csrfToken: string): Promise<unknown> =>
  del(`/v1/provider-credentials/${id}`, voidSchema, csrfToken);

export const rotateProviderCredential = (
  id: string,
  secret: string,
  csrfToken: string,
): Promise<ProviderCredential> =>
  post(`/v1/provider-credentials/${id}/rotate`, { secret }, providerCredentialSchema, csrfToken);

export const listModels = (): Promise<{ items: Model[] }> => get("/v1/models", modelsList);

export const syncProviderModels = (
  providerKind: string,
  apiKey: string,
  csrfToken: string,
): Promise<{ items: Model[] }> =>
  post(
    "/v1/models/sync",
    { provider_kind: providerKind, api_key: apiKey },
    modelsList,
    csrfToken,
  );

export const listProviderHealth = (): Promise<{ items: ProviderHealth[] }> =>
  get("/v1/provider-health", providerHealthList);

export const listActivity = (): Promise<{ data: ActivityEvent[] }> =>
  get("/v1/activity", activityList);

export function subscribeToActivity(
  onMessage: (event: ActivityEvent) => void,
  onError: (error: any) => void
): () => void {
  const source = new EventSource(`${CONTROL_API}/v1/activity/stream`, {
    withCredentials: true,
  });

  source.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data);
      onMessage(data);
    } catch (err) {
      onError(err);
    }
  };

  source.onerror = (e) => {
    onError(e);
  };

  return () => {
    source.close();
  };
}
