import type { ListResponse, RequestOptions, Transport } from "./client";

/** Control-plane API key. The key hash is never exposed; `key_prefix` is the only hint. */
export interface Credential {
  id: string;
  organization_id: string;
  name: string;
  key_prefix: string;
  scope: string;
  status: string;
  service_account_id?: string;
  last_used_at?: string;
  created_at: string;
  updated_at: string;
}

/** The raw control-plane key, returned exactly once. */
export interface CreatedCredential {
  credential: Credential;
  api_key: string;
}

/**
 * Gateway credential. On `create` and `rotate` the server returns this same flat
 * DTO with `key_prefix` holding the *raw* key, so read it immediately; every
 * other response carries only the prefix.
 */
export interface GatewayCredential {
  id: string;
  organization_id: string;
  name: string;
  key_prefix: string;
  subject_type: string;
  subject_id: string;
  team_id?: string;
  project_id?: string;
  scopes: string[];
  status: string;
  expires_at?: string;
  last_used_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateGatewayCredentialInput {
  name: string;
  subject_type: string;
  subject_id: string;
  team_id?: string;
  project_id?: string;
  scopes: string[];
  expires_at?: string;
}

export interface RotateGatewayCredentialInput {
  name?: string;
  scopes?: string[];
  expires_at?: string;
}

export interface CredentialsModule {
  create(
    input: { name: string; scope: string; service_account_id?: string },
    options?: RequestOptions,
  ): Promise<CreatedCredential>;
  list(options?: RequestOptions): Promise<ListResponse<Credential>>;
  remove(credentialId: string, options?: RequestOptions): Promise<void>;

  createGateway(
    input: CreateGatewayCredentialInput,
    options?: RequestOptions,
  ): Promise<GatewayCredential>;
  listGateway(options?: RequestOptions): Promise<ListResponse<GatewayCredential>>;
  getGateway(credentialId: string, options?: RequestOptions): Promise<GatewayCredential>;
  rotateGateway(
    credentialId: string,
    input: RotateGatewayCredentialInput,
    options?: RequestOptions,
  ): Promise<GatewayCredential>;
  expireGateway(credentialId: string, options?: RequestOptions): Promise<void>;
  restoreGateway(credentialId: string, options?: RequestOptions): Promise<void>;
  removeGateway(credentialId: string, options?: RequestOptions): Promise<void>;
}

export function createCredentialsModule(transport: Transport): CredentialsModule {
  return {
    create: (input, options) =>
      transport.request<CreatedCredential>("/v1/credentials", { ...options, method: "POST", body: input }),
    list: (options) => transport.request<ListResponse<Credential>>("/v1/credentials", options),
    remove: (credentialId, options) =>
      transport.requestAt<void>(`/v1/credentials/${encodeURIComponent(credentialId)}`, {
        ...options,
        method: "DELETE",
      }),

    createGateway: (input, options) =>
      transport.request<GatewayCredential>("/v1/gateway-credentials", {
        ...options,
        method: "POST",
        body: input,
      }),
    listGateway: (options) =>
      transport.request<ListResponse<GatewayCredential>>("/v1/gateway-credentials", options),
    getGateway: (credentialId, options) =>
      transport.requestAt<GatewayCredential>(
        `/v1/gateway-credentials/${encodeURIComponent(credentialId)}`,
        options,
      ),
    rotateGateway: (credentialId, input, options) =>
      transport.requestAt<GatewayCredential>(
        `/v1/gateway-credentials/${encodeURIComponent(credentialId)}/rotate`,
        { ...options, method: "POST", body: input },
      ),
    expireGateway: (credentialId, options) =>
      transport.requestAt<void>(`/v1/gateway-credentials/${encodeURIComponent(credentialId)}/expire`, {
        ...options,
        method: "POST",
      }),
    restoreGateway: (credentialId, options) =>
      transport.requestAt<void>(`/v1/gateway-credentials/${encodeURIComponent(credentialId)}/restore`, {
        ...options,
        method: "POST",
      }),
    removeGateway: (credentialId, options) =>
      transport.requestAt<void>(`/v1/gateway-credentials/${encodeURIComponent(credentialId)}`, {
        ...options,
        method: "DELETE",
      }),
  };
}