"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

export interface AcceptInviteState {
  readonly error?: string;
  readonly success?: boolean;
}

const origin = () => process.env.CONTROL_API_URL || "http://localhost:8081";

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

async function getCookieHeader(): Promise<string> {
  const jar = await cookies();
  return jar
    .getAll()
    .map((c) => `${c.name}=${c.value}`)
    .join("; ");
}

export async function acceptInvitationOnly(token: string): Promise<AcceptInviteState> {
  if (!token) return { error: "Invitation token is missing." };

  const cookieHeader = await getCookieHeader();
  const jar = await cookies();
  const sessionCookie = process.env.AIGW_AUTH_SESSION_COOKIE_NAME || "session";
  const csrfToken = jar.get(`${sessionCookie}_csrf`)?.value || "";

  try {
    const res = await fetch(new URL("/v1/invitations/accept", origin()), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Cookie: cookieHeader,
        "X-CSRF-Token": csrfToken,
      },
      body: JSON.stringify({ token }),
      cache: "no-store",
    });

    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      return { error: data.error?.message || "Failed to accept invitation." };
    }
  } catch {
    return { error: "Cannot reach the control API." };
  }

  redirect("/app/onboarding");
}

export async function authAndAcceptInvitation(
  token: string,
  mode: "signin" | "signup",
  formData: FormData
): Promise<AcceptInviteState> {
  if (!token) return { error: "Invitation token is required." };

  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");

  if (!email || !password) return { error: "Email and password are required." };

  let authEndpoint = "/v1/auth/signin";
  let authBody: Record<string, string> = { email, password };

  if (mode === "signup") {
    const displayName = String(formData.get("display_name") ?? "").trim() || email.split("@")[0];
    const organizationName = String(formData.get("organization_name") ?? "").trim() || "My Workspace";
    authEndpoint = "/v1/auth/signup";
    authBody = { email, password, display_name: displayName, organization_name: organizationName };
  }

  let authRes: Response;
  try {
    authRes = await fetch(new URL(authEndpoint, origin()), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(authBody),
      cache: "no-store",
    });
  } catch {
    return { error: "Cannot reach the control API." };
  }

  if (!authRes.ok) {
    const data = await authRes.json().catch(() => ({}));
    return { error: data.error?.message || "Authentication failed." };
  }

  await forwardCookies(authRes);

  // Now accept invitation
  const sessionCookie = process.env.AIGW_AUTH_SESSION_COOKIE_NAME || "session";
  const jar = await cookies();
  const csrfToken = jar.get(`${sessionCookie}_csrf`)?.value || "";
  const cookieHeader = await getCookieHeader();

  try {
    const acceptRes = await fetch(new URL("/v1/invitations/accept", origin()), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Cookie: cookieHeader,
        "X-CSRF-Token": csrfToken,
      },
      body: JSON.stringify({ token }),
      cache: "no-store",
    });

    if (!acceptRes.ok) {
      const data = await acceptRes.json().catch(() => ({}));
      return { error: data.error?.message || "Failed to accept invitation after authentication." };
    }
  } catch {
    return { error: "Could not finalize invitation acceptance." };
  }

  redirect("/app/onboarding");
}
