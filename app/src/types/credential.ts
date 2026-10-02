import type { ID, ISODateString, StatusTone } from "./common";

export type CredentialStatus = "healthy" | "expiring" | "expired" | "revoked";

export type CredentialKind = "api_key" | "oauth" | "service_account" | "webhook";

export interface Credential {
  id: ID;
  name: string;
  kind: CredentialKind;
  providerId?: ID;
  providerName: string;
  status: CredentialStatus;
  /** Last four characters only; the secret itself is never returned. */
  maskedValue: string;
  createdAt: ISODateString;
  expiresAt?: ISODateString;
  lastUsedAt: ISODateString;
  /** Number of distinct projects reading this credential. */
  usedByProjectCount: number;
}

export const CREDENTIAL_STATUS_TONE: Record<CredentialStatus, StatusTone> = {
  healthy: "success",
  expiring: "warning",
  expired: "danger",
  revoked: "neutral",
};
