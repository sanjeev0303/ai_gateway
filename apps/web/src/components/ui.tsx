import type { ReactNode } from "react";

const dateFormat = new Intl.DateTimeFormat("en-CA", {
  timeZone: "UTC",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
});

const dateTimeFormat = new Intl.DateTimeFormat("en-CA", {
  timeZone: "UTC",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

export function formatDate(value: string | undefined): string {
  if (!value) {
    return "—";
  }
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? "—" : `${dateFormat.format(parsed)} UTC`;
}

export function formatDateTime(value: string | undefined): string {
  if (!value) {
    return "—";
  }
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? "—" : `${dateTimeFormat.format(parsed)} UTC`;
}

export function never(value: string | undefined, label: string): string {
  return value ? formatDate(value) : label;
}

const GOOD = new Set(["active", "succeeded", "healthy", "ok", "ready", "enabled"]);
const BAD = new Set(["revoked", "failed", "unhealthy", "expired", "cancelled", "inactive"]);

function tone(value: string): string {
  const v = value.toLowerCase();
  if (GOOD.has(v)) {
    return "ok";
  }
  if (BAD.has(v)) {
    return "bad";
  }
  return "neutral";
}

export function StatusPill({ value }: { readonly value: string }) {
  return <span className={`pill pill-${tone(value)}`}>{value.replace(/_/g, " ")}</span>;
}

export function Mono({ children }: { readonly children: ReactNode }) {
  return <code className="mono">{children}</code>;
}

export function EmptyNote({ children }: { readonly children: ReactNode }) {
  return <p className="note note-empty">{children}</p>;
}

export function ErrorNote({ error }: { readonly error: unknown }) {
  const message = error instanceof Error ? error.message : "Request failed";
  return (
    <p className="note note-error" role="alert">
      {message}
    </p>
  );
}

export function LoadingRows({ columns }: { readonly columns: number }) {
  return (
    <tr aria-hidden>
      {Array.from({ length: columns }, (_, i) => (
        <td key={i} className="cell-skeleton" />
      ))}
    </tr>
  );
}

export interface Column<TRow> {
  readonly key: string;
  readonly header: string;
  readonly render: (row: TRow) => ReactNode;
  readonly align?: "left" | "right";
}

interface DataTableProps<TRow> {
  readonly columns: readonly Column<TRow>[];
  readonly rows: readonly TRow[];
  readonly isLoading: boolean;
  readonly isError: boolean;
  readonly error: unknown;
  readonly empty: string;
  readonly caption: string;
}

export function DataTable<TRow>({
  columns,
  rows,
  isLoading,
  isError,
  error,
  empty,
  caption,
}: DataTableProps<TRow>) {
  if (isError) {
    return <ErrorNote error={error} />;
  }
  return (
    <div className="table-wrap">
      <table className="table">
        <caption className="sr-only">{caption}</caption>
        <thead>
          <tr>
            {columns.map((column) => (
              <th key={column.key} scope="col" data-align={column.align ?? "left"}>
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {isLoading ? (
            <>
              <LoadingRows columns={columns.length} />
              <LoadingRows columns={columns.length} />
              <LoadingRows columns={columns.length} />
            </>
          ) : rows.length === 0 ? (
            <tr>
              <td colSpan={columns.length}>
                <EmptyNote>{empty}</EmptyNote>
              </td>
            </tr>
          ) : (
            rows.map((row, index) => (
              <tr key={index}>
                {columns.map((column) => (
                  <td key={column.key} data-align={column.align ?? "left"}>
                    {column.render(row)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}

export function Field({
  label,
  error,
  children,
}: {
  readonly label: string;
  readonly error?: string | undefined;
  readonly children: ReactNode;
}) {
  return (
    <label className="field">
      <span>{label}</span>
      {children}
      {error !== undefined && error !== "" && <span className="field-error">{error}</span>}
    </label>
  );
}

export function SubmitButton({
  pending,
  label,
  pendingLabel,
}: {
  readonly pending: boolean;
  readonly label: string;
  readonly pendingLabel: string;
}) {
  return (
    <button type="submit" className="btn" disabled={pending}>
      {pending ? pendingLabel : label}
    </button>
  );
}

export function SecretReveal({
  label,
  value,
  onDismiss,
}: {
  readonly label: string;
  readonly value: string;
  readonly onDismiss: () => void;
}) {
  return (
    <div className="reveal" role="status">
      <p className="reveal-title">
        {label} — copy it now. It is not stored and cannot be shown again.
      </p>
      <code className="reveal-value">{value}</code>
      <button type="button" className="btn btn-ghost" onClick={onDismiss}>
        I have saved it
      </button>
    </div>
  );
}

export function MetricGrid({ children }: { readonly children: ReactNode }) {
  return <div className="metric-grid">{children}</div>;
}

export function MetricCard({
  title,
  value,
  trend,
  trendLabel,
  icon,
}: {
  readonly title: string;
  readonly value: ReactNode;
  readonly trend?: "up" | "down" | "neutral";
  readonly trendLabel?: string;
  readonly icon?: ReactNode;
}) {
  return (
    <div className="card metric-card">
      <div className="flex justify-between items-center metric-label">
        <span>{title}</span>
        {icon && <span>{icon}</span>}
      </div>
      <div className="metric-value">{value}</div>
      {(trend || trendLabel) && (
        <div className="text-xs mt-1 flex items-center gap-1">
          {trend === "up" && <span className="text-status-healthy">↑</span>}
          {trend === "down" && <span className="text-status-down">↓</span>}
          {trend === "neutral" && <span className="text-muted">→</span>}
          <span className="text-muted">{trendLabel}</span>
        </div>
      )}
    </div>
  );
}

export function PageHeader({
  title,
  description,
  children,
}: {
  readonly title: string;
  readonly description?: string;
  readonly children?: ReactNode;
}) {
  return (
    <div className="page-header">
      <div>
        <h1>{title}</h1>
        {description && <p>{description}</p>}
      </div>
      {children && <div className="page-header-actions">{children}</div>}
    </div>
  );
}

export function StatusBadge({
  status,
  label,
}: {
  readonly status: "success" | "warning" | "error" | "info" | "neutral";
  readonly label?: string;
}) {
  const styles = {
    success: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20",
    warning: "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20",
    error: "bg-rose-500/10 text-rose-600 dark:text-rose-400 border-rose-500/20",
    info: "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20",
    neutral: "bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20",
  };
  
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium border ${styles[status]}`}>
      {label ?? status}
    </span>
  );
}

export function HealthIndicator({
  health,
}: {
  readonly health: "healthy" | "degraded" | "unavailable" | "unknown";
}) {
  const styles = {
    healthy: "bg-emerald-500",
    degraded: "bg-amber-500",
    unavailable: "bg-rose-500",
    unknown: "bg-gray-400",
  };
  return (
    <div className="flex items-center gap-2">
      <span className={`w-2.5 h-2.5 rounded-full ${styles[health]}`} />
      <span className="text-sm font-medium capitalize">{health}</span>
    </div>
  );
}

export function EmptyState({
  title,
  description,
  action,
}: {
  readonly title: string;
  readonly description?: string;
  readonly action?: ReactNode;
}) {
  return (
    <div className="empty-state">
      <div className="empty-icon">∅</div>
      <h3>{title}</h3>
      {description && <p>{description}</p>}
      {action && <div className="empty-action">{action}</div>}
    </div>
  );
}

export function ErrorState({
  title,
  error,
  retry,
}: {
  readonly title: string;
  readonly error?: unknown;
  readonly retry?: () => void;
}) {
  const message = error instanceof Error ? error.message : typeof error === "string" ? error : "An unexpected error occurred.";
  return (
    <div className="error-state">
      <div className="error-icon">!</div>
      <h3>{title}</h3>
      {error !== undefined && <p>{message}</p>}
      {retry && (
        <button onClick={retry} className="btn mt-4">
          Try Again
        </button>
      )}
    </div>
  );
}

export function Skeleton({ className }: { readonly className?: string }) {
  return <div className={`animate-pulse bg-muted rounded ${className || "h-4 w-full"}`} />;
}

export function LoadingState({ text }: { readonly text?: string }) {
  return (
    <div className="loading-state">
      <div className="spinner" />
      <p>{text ?? "Loading..."}</p>
    </div>
  );
}

export function AppShell({
  sidebar,
  topbar,
  children,
}: {
  readonly sidebar: ReactNode;
  readonly topbar: ReactNode;
  readonly children: ReactNode;
}) {
  return (
    <div className="app-shell">
      <aside className="sidebar">
        {sidebar}
      </aside>
      <div className="main-content">
        <header className="topbar">
          {topbar}
        </header>
        <main className="page-content">
          {children}
        </main>
      </div>
    </div>
  );
}

export function CodeBlock({ code, language }: { readonly code: string; readonly language?: string }) {
  return (
    <pre className="bg-muted/30 p-4 rounded-lg overflow-x-auto text-sm font-mono border border-muted">
      <code data-language={language}>{code}</code>
    </pre>
  );
}

export function Sidebar({ children }: { readonly children: ReactNode }) {
  return <div className="sidebar-nav">{children}</div>;
}

export function Topbar({ children }: { readonly children: ReactNode }) {
  return <div className="flex items-center justify-between" style={{width: "100%"}}>{children}</div>;
}

export function Breadcrumbs({ crumbs }: { readonly crumbs: { label: string; href?: string }[] }) {
  return (
    <nav className="flex items-center space-x-2 text-sm text-muted-foreground">
      {crumbs.map((crumb, i) => (
        <span key={i} className="flex items-center">
          {crumb.href ? <a href={crumb.href} className="hover:underline">{crumb.label}</a> : crumb.label}
          {i < crumbs.length - 1 && <span className="mx-2">/</span>}
        </span>
      ))}
    </nav>
  );
}

export function PageToolbar({ children }: { readonly children: ReactNode }) {
  return <div className="flex items-center space-x-2 mb-4">{children}</div>;
}

export function DataTableToolbar({ children }: { readonly children: ReactNode }) {
  return <div className="flex items-center justify-between mb-4">{children}</div>;
}

export function Pagination({ current, total, onPageChange }: { readonly current: number; readonly total: number; readonly onPageChange?: (page: number) => void }) {
  return (
    <div className="flex items-center space-x-2 mt-4 text-sm">
      <button disabled={current <= 1} onClick={() => onPageChange?.(current - 1)} className="btn btn-outline btn-sm">Prev</button>
      <span>Page {current} of {total}</span>
      <button disabled={current >= total} onClick={() => onPageChange?.(current + 1)} className="btn btn-outline btn-sm">Next</button>
    </div>
  );
}

export function ChartContainer({ children, className }: { readonly children: ReactNode; readonly className?: string }) {
  return <div className={`border rounded-lg p-4 bg-card text-card-foreground shadow-sm ${className || ''}`}>{children}</div>;
}

export function ChartLegend({ items }: { readonly items: { color: string; label: string }[] }) {
  return (
    <div className="flex flex-wrap items-center gap-4 text-sm mt-4">
      {items.map((item, i) => (
        <div key={i} className="flex items-center space-x-2">
          <div className="w-3 h-3 rounded-full" style={{ backgroundColor: item.color }} />
          <span>{item.label}</span>
        </div>
      ))}
    </div>
  );
}

export function FilterBar({ children }: { readonly children: ReactNode }) {
  return <div className="flex items-center space-x-4 p-4 border-b bg-muted/10">{children}</div>;
}

export function DateRangePicker() {
  return <div className="text-sm border rounded px-3 py-1.5 cursor-pointer bg-background">Select Date Range</div>;
}

export function SearchInput({ placeholder, value, onChange }: { readonly placeholder?: string; readonly value?: string; readonly onChange?: (val: string) => void }) {
  return (
    <input 
      type="text" 
      className="border rounded px-3 py-1.5 text-sm bg-background w-full max-w-sm" 
      placeholder={placeholder ?? "Search..."} 
      value={value}
      onChange={(e) => onChange?.(e.target.value)}
    />
  );
}

export function Select({ options, value, onChange }: { readonly options: { label: string; value: string }[]; readonly value?: string; readonly onChange?: (val: string) => void }) {
  return (
    <select className="border rounded px-3 py-1.5 text-sm bg-background" value={value} onChange={(e) => onChange?.(e.target.value)}>
      {options.map(opt => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
    </select>
  );
}

export function Tabs({ tabs, active, onChange }: { readonly tabs: string[]; readonly active: string; readonly onChange?: (tab: string) => void }) {
  return (
    <div className="flex space-x-4 border-b border-muted">
      {tabs.map(tab => (
        <button 
          key={tab} 
          className={`py-2 px-1 border-b-2 text-sm font-medium ${active === tab ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
          onClick={() => onChange?.(tab)}
        >
          {tab}
        </button>
      ))}
    </div>
  );
}

export function Drawer({ open, onClose, children }: { readonly open: boolean; readonly onClose: () => void; readonly children: ReactNode }) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="w-full max-w-md bg-background h-full shadow-lg relative p-6 flex flex-col">
        <button onClick={onClose} className="absolute top-4 right-4">&times;</button>
        {children}
      </div>
    </div>
  );
}

export function Modal({ open, onClose, children }: { readonly open: boolean; readonly onClose: () => void; readonly children: ReactNode }) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="bg-background rounded-lg shadow-lg relative p-6 max-w-lg w-full">
        {children}
      </div>
    </div>
  );
}

export function ConfirmDialog({ open, onConfirm, onCancel, title, description }: { readonly open: boolean; readonly onConfirm: () => void; readonly onCancel: () => void; readonly title: string; readonly description: string }) {
  return (
    <Modal open={open} onClose={onCancel}>
      <h3 className="text-lg font-semibold">{title}</h3>
      <p className="text-muted-foreground mt-2">{description}</p>
      <div className="flex justify-end space-x-2 mt-6">
        <button onClick={onCancel} className="btn btn-outline">Cancel</button>
        <button onClick={onConfirm} className="btn btn-danger">Confirm</button>
      </div>
    </Modal>
  );
}

export function Toast({ message, type = "info" }: { readonly message: string; readonly type?: "info" | "success" | "error" }) {
  const styles = {
    info: "bg-blue-500",
    success: "bg-emerald-500",
    error: "bg-rose-500",
  };
  return (
    <div className={`fixed bottom-4 right-4 text-white px-4 py-2 rounded shadow-lg text-sm ${styles[type]}`}>
      {message}
    </div>
  );
}

export function CopyButton({ text }: { readonly text: string }) {
  return <button className="btn btn-ghost btn-sm" onClick={() => navigator.clipboard.writeText(text)}>Copy</button>;
}

export function UsageBar({ percent }: { readonly percent: number }) {
  return (
    <div className="w-full bg-muted rounded-full h-2 overflow-hidden">
      <div className="bg-primary h-full" style={{ width: `${Math.min(100, Math.max(0, percent))}%` }} />
    </div>
  );
}

export function ProgressBar({ progress }: { readonly progress: number }) {
  return <UsageBar percent={progress} />;
}

export function Avatar({ initials }: { readonly initials: string }) {
  return <div className="w-8 h-8 rounded-full bg-primary/20 text-primary flex items-center justify-center text-xs font-semibold">{initials}</div>;
}

export function UserPresence({ name, online }: { readonly name: string; readonly online: boolean }) {
  return (
    <div className="flex items-center space-x-2">
      <div className="relative">
        <Avatar initials={name.substring(0, 2).toUpperCase()} />
        {online && <span className="absolute bottom-0 right-0 w-2.5 h-2.5 bg-emerald-500 border-2 border-background rounded-full" />}
      </div>
      <span className="text-sm font-medium">{name}</span>
    </div>
  );
}

export function ProviderBadge({ provider }: { readonly provider: string }) {
  return <StatusBadge status="neutral" label={provider} />;
}

export function ModelBadge({ model }: { readonly model: string }) {
  return <span className="inline-flex items-center rounded-md bg-muted px-2 py-1 text-xs font-medium text-muted-foreground ring-1 ring-inset ring-muted-foreground/20">{model}</span>;
}

export function CostDisplay({ amount, currency = "USD" }: { readonly amount: number; readonly currency?: string }) {
  return <span className="font-mono text-sm tracking-tight">{new Intl.NumberFormat("en-US", { style: "currency", currency }).format(amount)}</span>;
}

export function TokenDisplay({ count }: { readonly count: number }) {
  return <span className="font-mono text-sm tracking-tight">{new Intl.NumberFormat("en-US").format(count)}</span>;
}
