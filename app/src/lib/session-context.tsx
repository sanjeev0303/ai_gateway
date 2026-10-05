"use client";

import * as React from "react";

import type { DashboardSession } from "@/types";

export type SessionState =
  | { status: "loading" }
  | { status: "authenticated"; session: DashboardSession }
  | { status: "unauthenticated" }
  | { status: "error"; error: string };

const SessionContext = React.createContext<SessionState | null>(null);

export function SessionProvider({
  session,
  children,
}: {
  session: DashboardSession | null;
  children: React.ReactNode;
}) {
  const state: SessionState = session
    ? { status: "authenticated", session }
    : { status: "unauthenticated" };
  return <SessionContext.Provider value={state}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionState {
  const value = React.useContext(SessionContext);
  if (!value) {
    return { status: "loading" };
  }
  return value;
}
