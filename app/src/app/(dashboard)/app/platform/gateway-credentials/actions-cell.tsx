"use client";

import { GatewayCredentialActions } from "./client";
import type { CredentialRow } from "./page";

export function CredentialActionsCell({ row }: { row: CredentialRow }) {
  return <GatewayCredentialActions credential={row} />;
}