import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { createApiClient, type AdminMe, type ApiClient } from "@ai-governance/api-client";

// Must match the backend's AIGW_AUTH_SESSION_COOKIE_NAME.
const SESSION_COOKIE = process.env.AIGW_AUTH_SESSION_COOKIE_NAME || "session";
const CSRF_COOKIE = `${SESSION_COOKIE}_csrf`;

/** Absolute control-API origin for server-side calls (not the browser proxy). */
export const controlApiOrigin = () => process.env.CONTROL_API_URL || "http://localhost:8081";

/** A client bound to the caller's session cookies. */
export async function getApiClient(): Promise<ApiClient> {
  const jar = await cookies();
  return createApiClient({
    baseUrl: controlApiOrigin(),
    cookie: jar.toString(),
    csrfToken: jar.get(CSRF_COOKIE)?.value,
  });
}

/**
 * The signed-in platform admin, or null. The control API answers 404 to
 * anyone who is not one, so any failure means "not an admin".
 */
export async function getPlatformAdmin(): Promise<AdminMe | null> {
  const jar = await cookies();
  if (!jar.has(SESSION_COOKIE)) return null;
  try {
    const { admin } = await getApiClient();
    return await admin.me();
  } catch {
    return null;
  }
}

/**
 * Guards every console page. Without a session the visitor signs in; a
 * signed-in account that is not on the allowlist is told so on the sign-in
 * page instead of seeing a blank 404.
 */
export async function requirePlatformAdmin(): Promise<AdminMe> {
  const jar = await cookies();
  if (!jar.has(SESSION_COOKIE)) redirect("/signin");
  const admin = await getPlatformAdmin();
  if (!admin) redirect("/signin?reason=not-admin");
  return admin;
}

export const sessionCookieNames = [SESSION_COOKIE, CSRF_COOKIE, `${SESSION_COOKIE}_refresh`];
