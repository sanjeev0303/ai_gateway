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
