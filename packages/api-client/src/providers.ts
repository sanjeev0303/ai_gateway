import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";
import type { Model } from "./models";

export type ProviderAccount = components["schemas"]["ProviderAccount"];
/** Never carries the secret itself, only a fingerprint and last characters. */
export type ProviderCredential = components["schemas"]["ProviderCredential"];
export type ProviderHealth = components["schemas"]["ProviderHealth"];
export type CreateProviderAccountInput = components["schemas"]["CreateProviderAccountRequest"];
export type UpdateProviderAccountInput = components["schemas"]["UpdateProviderAccountRequest"];
export type CreateProviderCredentialInput = components["schemas"]["CreateProviderCredentialRequest"];
export type RotateProviderCredentialInput = components["schemas"]["RotateProviderCredentialRequest"];
export type ProviderValidationResult = components["schemas"]["ProviderValidationResult"];

/**
 * A live probe. The secret is used for the upstream call only: it is never
 * stored, returned, or logged. Persist a credential instead if you need it kept.
 */
export type ProviderProbeInput = components["schemas"]["ProviderProbeRequest"];

export interface ProvidersModule {
  createAccount(
    input: CreateProviderAccountInput,
    options?: RequestOptions,
  ): Promise<ProviderAccount>;
  listAccounts(options?: RequestOptions): Promise<ListResponse<ProviderAccount>>;
  getAccount(accountId: string, options?: RequestOptions): Promise<ProviderAccount>;
  updateAccount(
    accountId: string,
    input: UpdateProviderAccountInput,
    options?: RequestOptions,
  ): Promise<ProviderAccount>;
  deleteAccount(accountId: string, options?: RequestOptions): Promise<void>;
  validateAccount(accountId: string, options?: RequestOptions): Promise<ProviderValidationResult>;
  checkAccountHealth(accountId: string, options?: RequestOptions): Promise<ProviderHealth>;
  syncAccountModels(accountId: string, options?: RequestOptions): Promise<ListResponse<Model>>;

  createCredential(
    accountId: string,
    input: CreateProviderCredentialInput,
    options?: RequestOptions,
  ): Promise<ProviderCredential>;
  listCredentials(options?: RequestOptions): Promise<ListResponse<ProviderCredential>>;
  getCredential(credentialId: string, options?: RequestOptions): Promise<ProviderCredential>;
  rotateCredential(
    credentialId: string,
    input: RotateProviderCredentialInput,
    options?: RequestOptions,
  ): Promise<ProviderCredential>;
  removeCredential(credentialId: string, options?: RequestOptions): Promise<void>;
  validate(input: ProviderProbeInput, options?: RequestOptions): Promise<ProviderValidationResult>;

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
    updateAccount: (accountId, input, options) =>
      transport.requestAt<ProviderAccount>(`/v1/provider-accounts/${encodeURIComponent(accountId)}`, {
        ...options,
        method: "PATCH",
        body: input,
      }),
    deleteAccount: (accountId, options) =>
      transport.requestAt<void>(`/v1/provider-accounts/${encodeURIComponent(accountId)}`, {
        ...options,
        method: "DELETE",
      }),
    validateAccount: (accountId, options) =>
      transport.requestAt<ProviderValidationResult>(
        `/v1/provider-accounts/${encodeURIComponent(accountId)}/validate`,
        { ...options, method: "POST" },
      ),
    checkAccountHealth: (accountId, options) =>
      transport.requestAt<ProviderHealth>(
        `/v1/provider-accounts/${encodeURIComponent(accountId)}/health`,
        { ...options, method: "POST" },
      ),
    syncAccountModels: (accountId, options) =>
      transport.requestAt<ListResponse<Model>>(
        `/v1/provider-accounts/${encodeURIComponent(accountId)}/sync`,
        { ...options, method: "POST" },
      ),

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
      transport.request<ProviderValidationResult>("/v1/provider-credentials/validate", {
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