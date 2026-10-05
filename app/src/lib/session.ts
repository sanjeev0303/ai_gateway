import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import {
  ApiError,
  createApiClient,
  type ApiClient,
  type Session,
} from "@ai-governance/api-client";

import type { DashboardSession, SessionOrganization } from "@/types";

/**
 * Server-side session access.
 *
 * The backend issues the session cookie on its own origin; Server Components
 * have no ambient cookie jar, so the browser's `Cookie` header is forwarded
 * verbatim on every call. Mutations additionally echo the CSRF cookie in
 * `X-CSRF-Token` (double submit). Both cookie names mirror the backend's
 * `AIGW_AUTH_SESSION_COOKIE_NAME` setting.
 */

// Keep in sync with the backend's AIGW_AUTH_SESSION_COOKIE_NAME default.
const SESSION_COOKIE = process.env.AIGW_AUTH_SESSION_COOKIE_NAME || "session";
const CSRF_COOKIE = `${SESSION_COOKIE}_csrf`;

/**
 * Absolute control-API origin for server-side calls. A relative base would
 * throw in `new URL()`, and Server Components are not subject to CORS, so they
 * bypass the same-origin proxy that the browser uses.
 */
const origin = () => process.env.CONTROL_API_URL || "http://localhost:8081";

/** A client bound to the caller's session. Each call carries its own cookies. */
export async function getApiClient(): Promise<ApiClient> {
  const jar = await cookies();
  return createApiClient({
    baseUrl: origin(),
    cookie: jar.toString(),
    csrfToken: jar.get(CSRF_COOKIE)?.value,
  });
}

/**
 * The current session, or null when unauthenticated. A 401 is the expected
 * "no session" answer rather than a failure, so it is not thrown; any other
 * error also yields null so pages degrade instead of crashing.
 */
export async function getSession(): Promise<Session | null> {
  try {
    const { auth } = await getApiClient();
    return await auth.session();
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
      return null;
    }
    return null;
  }
}

/** Session or redirect to sign-in. Use at the top of authenticated pages. */
export async function requireSession(): Promise<Session> {
  const session = await getSession();
  if (!session) redirect("/signin");
  return session;
}

/**
 * The caller's organization, resolved from their session. Callers are already
 * authenticated, so a failure here is a real error and is allowed to propagate
 * rather than degrading to null the way the session probe does.
 *
 * The API is snake_case; `SessionOrganization` is camelCase, so the mapping is
 * explicit here rather than an unchecked cast at each call site.
 */
export async function getCurrentOrganization(): Promise<SessionOrganization> {
  const { organization } = await getApiClient();
  const org = await organization.get();
  return {
    id: org.id,
    name: org.name,
    slug: org.slug,
    status: org.status,
    createdAt: org.created_at,
    updatedAt: org.updated_at,
  };
}

/**
 * Everything the dashboard chrome needs: session identity plus the
 * organization row. Redirects to sign-in when there is no session.
 */
export async function requireDashboardSession(): Promise<DashboardSession> {
  const { claims } = await requireSession();
  const organization = await getCurrentOrganization();
  return {
    user: {
      name: claims.display_name || claims.email,
      email: claims.email,
      role: claims.role,
    },
    organization,
  };
}