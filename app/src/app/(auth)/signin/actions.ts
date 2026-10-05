"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

/**
 * Sign-in actions.
 *
 * These run on the server, so they call the control API directly instead of
 * going through the same-origin proxy. The JWT only ever travels in the
 * HttpOnly cookie, never in the JSON body, so the backend's Set-Cookie headers
 * are replayed onto the response rather than reconstructed from the payload.
 */

export interface AuthState {
  readonly error?: string;
}

const origin = () => process.env.CONTROL_API_URL || "http://localhost:8081";

// Keep in sync with the backend's AIGW_AUTH_SESSION_COOKIE_NAME default. The
// backend derives the refresh and CSRF cookie names from that same prefix
// (controlapi/auth.go), so all three names track one env var.
const SESSION_COOKIE = process.env.AIGW_AUTH_SESSION_COOKIE_NAME || "session";
const CSRF_COOKIE = `${SESSION_COOKIE}_csrf`;
const REFRESH_COOKIE = `${SESSION_COOKIE}_refresh`;

/** Replay the backend's cookies onto this origin, preserving its attributes. */
async function forwardCookies(response: Response): Promise<void> {
  const jar = await cookies();
  for (const header of response.headers.getSetCookie()) {
    const [pair, ...attributes] = header.split(";");
    const separator = pair.indexOf("=");
    if (separator < 0) continue;

    const options: Parameters<typeof jar.set>[2] = { path: "/" };
    for (const attribute of attributes) {
      const [rawKey, rawValue] = attribute.split("=");
      const key = rawKey.trim().toLowerCase();
      const value = rawValue?.trim();
      switch (key) {
        case "path":
          options.path = value || "/";
          break;
        case "httponly":
          options.httpOnly = true;
          break;
        case "secure":
          options.secure = true;
          break;
        case "samesite":
          options.sameSite = value?.toLowerCase() as "strict" | "lax" | "none";
          break;
        case "max-age":
          options.maxAge = Number(value);
          break;
      }
    }
    jar.set(pair.slice(0, separator).trim(), pair.slice(separator + 1).trim(), options);
  }
}

async function messageFrom(response: Response, fallback: string): Promise<string> {
  try {
    const body = (await response.json()) as { error?: { message?: string } };
    return body.error?.message || fallback;
  } catch {
    return fallback;
  }
}

async function authenticate(
  path: string,
  body: unknown,
  fallback: string,
): Promise<AuthState> {
  let response: Response;
  try {
    response = await fetch(new URL(path, origin()), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      cache: "no-store",
    });
  } catch {
    // The control API being unreachable is an operator problem, not bad input,
    // so it gets a distinct message instead of implying a wrong password.
    return { error: "Cannot reach the control API." };
  }

  if (!response.ok) return { error: await messageFrom(response, fallback) };
  await forwardCookies(response);
  redirect("/app/overview");
}

export async function signIn(_state: AuthState, formData: FormData): Promise<AuthState> {
  return authenticate(
    "/v1/auth/signin",
    {
      email: String(formData.get("email") ?? ""),
      password: String(formData.get("password") ?? ""),
    },
    "Sign-in failed.",
  );
}

export async function signUp(_state: AuthState, formData: FormData): Promise<AuthState> {
  return authenticate(
    "/v1/auth/signup",
    {
      email: String(formData.get("email") ?? ""),
      display_name: String(formData.get("display_name") ?? ""),
      organization_name: String(formData.get("organization_name") ?? ""),
      password: String(formData.get("password") ?? ""),
    },
    "Sign-up failed.",
  );
}

/**
 * Passwordless local development sign-in. The backend provisions the account on
 * first use, which is how a fresh environment gets its first session.
 */
export async function devSignIn(
  _state: AuthState,
  formData: FormData,
): Promise<AuthState> {
  const email = String(formData.get("email") ?? "");
  return authenticate(
    "/v1/auth/dev-login",
    {
      email,
      display_name: email.split("@")[0] || "Developer",
    },
    "Development sign-in failed.",
  );
}

/**
 * Revoke the session, drop the local cookies, and return to sign-in. The API
 * call is best-effort so a local sign-out still completes when the control API
 * is unreachable.
 */
export async function signOut(): Promise<void> {
  const jar = await cookies();
  try {
    await fetch(new URL("/v1/auth/logout", origin()), {
      method: "POST",
      headers: { cookie: jar.toString() },
      cache: "no-store",
    });
  } catch {
    // Best-effort: the local cookies are cleared regardless.
  }
  jar.delete(SESSION_COOKIE);
  jar.delete(CSRF_COOKIE);
  jar.delete(REFRESH_COOKIE);
  redirect("/signin");
}