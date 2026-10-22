import { DashboardShell } from "@/components/dashboard/dashboard-shell";
import { SessionProvider } from "@/lib/session-context";
import { ThemeProvider } from "@/components/providers/theme-provider";
import { requireDashboardSession } from "@/lib/session";
import { redirect } from "next/navigation";

export default async function DashboardLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  let session: Awaited<ReturnType<typeof requireDashboardSession>>;
  try {
    session = await requireDashboardSession();
  } catch (error) {
    // Ignore Next.js internal redirect errors - they're not real failures.
    if (error instanceof Error && error.message === "NEXT_REDIRECT") {
      throw error;
    }
    // Log the actual error for server-side debugging.
    // React masks Server Component errors with a digest (#441) in production.
    console.error("[DashboardLayout] requireDashboardSession failed:", {
      error: error instanceof Error ? error.message : String(error),
      stack: error instanceof Error ? error.stack : undefined,
      timestamp: new Date().toISOString(),
    });
    // Redirect to sign-in on auth failures; let other errors propagate to error.tsx
    if (error instanceof Error && (error.message.includes("401") || error.message.includes("unauthorized"))) {
      redirect("/signin");
    }
    throw error;
  }

  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
      <SessionProvider session={session}>
        <DashboardShell>{children}</DashboardShell>
      </SessionProvider>
    </ThemeProvider>
  );
}
