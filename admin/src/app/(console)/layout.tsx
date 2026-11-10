import { requirePlatformAdmin } from "@/lib/session";
import { AdminShell } from "./_components/admin-shell";

/**
 * Every console page sits behind this guard: only users on the control API's
 * PLATFORM_ADMIN_EMAILS allowlist get in. Anyone else is sent to sign in.
 */
export default async function ConsoleLayout({ children }: { children: React.ReactNode }) {
  const admin = await requirePlatformAdmin();
  return <AdminShell adminEmail={admin.email}>{children}</AdminShell>;
}
