"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { controlApiOrigin, sessionCookieNames } from "@/lib/session";

/** Ends the session on the control API and clears this app's cookies. */
export async function signOut(): Promise<void> {
  const jar = await cookies();
  try {
    await fetch(new URL("/v1/auth/logout", controlApiOrigin()), {
      method: "POST",
      headers: { cookie: jar.toString() },
      cache: "no-store",
    });
  } catch {
    // Best effort: the cookies below are cleared either way.
  }
  for (const name of sessionCookieNames) jar.delete(name);
  redirect("/signin");
}
