import { SidebarNav } from "@/components/dashboard/sidebar";
import { Topbar } from "@/components/dashboard/topbar";
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
        <div className="flex h-screen overflow-hidden">
          <aside className="hidden w-60 shrink-0 border-r border-sidebar-border bg-sidebar lg:block">
            <SidebarNav />
          </aside>

          <div className="flex min-w-0 flex-1 flex-col">
            <Topbar />
            <main className="min-h-0 flex-1 overflow-y-auto page-scrollbar">{children}</main>
          </div>
        </div>
      </SessionProvider>
    </ThemeProvider>
  );
}
