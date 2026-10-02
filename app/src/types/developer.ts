import type { ID, ISODateString, StatusTone } from "./common";

export interface DeveloperApiKey {
  id: ID;
  name: string;
  /** Last four characters only; the secret is shown once at creation. */
  maskedKey: string;
  environment: "production" | "staging" | "development";
  scopes: string[];
  createdAt: ISODateString;
  lastUsedAt: ISODateString;
  expiresAt?: ISODateString;
  status: "active" | "revoked" | "expired";
  requestsLast24h: number;
}

export interface CodeSnippet {
  id: ID;
  label: string;
  language: "bash" | "typescript" | "python";
  code: string;
}

export interface SdkIntegration {
  id: ID;
  name: string;
  language: string;
  installed: boolean;
  latestVersion: string;
  docsUrl: string;
}

export const API_KEY_STATUS_TONE: Record<DeveloperApiKey["status"], StatusTone> = {
  active: "success",
  revoked: "neutral",
  expired: "danger",
};
