import type { ID, ISODateString } from "./common";

/**
 * Session-derived identity for the dashboard chrome.
 *
 * The session endpoint returns claims only (no organization name), so the
 * organization comes from `GET /v1/organizations/me`. Both shapes carry just
 * the fields the dashboard reads, not the full API payloads.
 *
 * Field names are camelCase even though the API is snake_case: `session.ts`
 * maps them, so components never touch raw API payloads.
 */

/** The backend constrains organization status to these two values. */
export type SessionOrganizationStatus = "active" | "suspended";

export interface SessionUser {
  name: string;
  email: string;
  role: string;
}

export interface SessionOrganization {
  id: ID;
  name: string;
  slug: string;
  status: SessionOrganizationStatus;
  createdAt: ISODateString;
  updatedAt: ISODateString;
}

export interface DashboardSession {
  user: SessionUser;
  organization: SessionOrganization;
}
