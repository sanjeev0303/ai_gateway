import type { ListResponse, RequestOptions, Transport } from "./client";

export interface ProviderAccount {
  id: string;
  organization_id: string;
  provider_kind: string;
  name: string;
  base_url?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

/** Never carries the secret itself, only a fingerprint and last characters. */
export interface ProviderCredential {
  id: string;
  organization_id: string;
  provider_account_id: string;
  name: string;
  kind: string;
  fingerprint: string;
  tail: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface ProviderHealth {
  provider_kind: string;
  status: string;
  status_reason?: string;
  last_error?: string;
  checked_at: string;
}

/**
 * A live probe. The secret is used for the upstream call only: it is never
 * stored, returned, or logged. Persist a credential instead if you need it kept.
 */
export interface ProviderProbeInput {
  provider_kind: string;
  api_key: string;
}

export interface ProvidersModule {
  createAccount(
    input: { provider_kind: string; name: string; base_url?: string },
    options?: RequestOptions,
  ): Promise<ProviderAccount>;
  listAccounts(options?: RequestOptions): Promise<ListResponse<ProviderAccount>>;
  getAccount(accountId: string, options?: RequestOptions): Promise<ProviderAccount>;

  createCredential(
    accountId: string,
    input: { name: string; kind: string; secret: string },
    options?: RequestOptions,
  ): Promise<ProviderCredential>;
  listCredentials(options?: RequestOptions): Promise<ListResponse<ProviderCredential>>;
  getCredential(credentialId: string, options?: RequestOptions): Promise<ProviderCredential>;
  rotateCredential(
    credentialId: string,
    input: { secret: string },
    options?: RequestOptions,
  ): Promise<ProviderCredential>;
  removeCredential(credentialId: string, options?: RequestOptions): Promise<void>;
  validate(input: ProviderProbeInput, options?: RequestOptions): Promise<{ valid: boolean }>;

  health(options?: RequestOptions): Promise<ListResponse<ProviderHealth>>;
  check(input: ProviderProbeInput, options?: RequestOptions): Promise<ProviderHealth>;
}

export function createProvidersModule(transport: Transport): ProvidersModule {
  return {
    createAccount: (input, options) =>
      transport.request<ProviderAccount>("/v1/provider-accounts", {
        ...options,
        method: "POST",
        body: input,
      }),
    listAccounts: (options) =>
      transport.request<ListResponse<ProviderAccount>>("/v1/provider-accounts", options),
    getAccount: (accountId, options) =>
      transport.requestAt<ProviderAccount>(`/v1/provider-accounts/${encodeURIComponent(accountId)}`, options),

    createCredential: (accountId, input, options) =>
      transport.requestAt<ProviderCredential>(
        `/v1/provider-accounts/${encodeURIComponent(accountId)}/credentials`,
        { ...options, method: "POST", body: input },
      ),
    listCredentials: (options) =>
      transport.request<ListResponse<ProviderCredential>>("/v1/provider-credentials", options),
    getCredential: (credentialId, options) =>
      transport.requestAt<ProviderCredential>(
        `/v1/provider-credentials/${encodeURIComponent(credentialId)}`,
        options,
      ),
    rotateCredential: (credentialId, input, options) =>
      transport.requestAt<ProviderCredential>(
        `/v1/provider-credentials/${encodeURIComponent(credentialId)}/rotate`,
        { ...options, method: "POST", body: input },
      ),
    removeCredential: (credentialId, options) =>
      transport.requestAt<void>(`/v1/provider-credentials/${encodeURIComponent(credentialId)}`, {
        ...options,
        method: "DELETE",
      }),
    validate: (input, options) =>
      transport.request<{ valid: boolean }>("/v1/provider-credentials/validate", {
        ...options,
        method: "POST",
        body: input,
      }),

    health: (options) => transport.request<ListResponse<ProviderHealth>>("/v1/provider-health", options),
    check: (input, options) =>
      transport.request<ProviderHealth>("/v1/provider-health/check", {
        ...options,
        method: "POST",
        body: input,
      }),
  };
}