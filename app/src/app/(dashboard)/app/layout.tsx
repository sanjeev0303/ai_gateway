import { SidebarNav } from "@/components/dashboard/sidebar";
import { Topbar } from "@/components/dashboard/topbar";
import { DemoDataProvider } from "@/lib/demo/store";

export default function DashboardLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <DemoDataProvider>
      <div className="flex h-screen overflow-hidden">
        <aside className="hidden w-60 shrink-0 border-r border-sidebar-border bg-sidebar lg:block">
          <SidebarNav />
        </aside>

        <div className="flex min-w-0 flex-1 flex-col">
          <Topbar />
          <main className="min-h-0 flex-1 overflow-y-auto">{children}</main>
        </div>
      </div>
    </DemoDataProvider>
  );
}
