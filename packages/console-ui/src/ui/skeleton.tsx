import { cn } from "cn"

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      className={cn("animate-pulse rounded-md bg-muted", className)}
      {...props}
    />
  )
}

function SkeletonText({ className, lines = 3, ...props }: React.ComponentProps<"div"> & { lines?: number }) {
  return (
    <div className={cn("space-y-2", className)} {...props}>
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton key={i} className="h-4 w-full" />
      ))}
    </div>
  )
}

function SkeletonCard({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div className={cn("rounded-lg border bg-card p-4 space-y-4", className)} {...props}>
      <div className="flex items-center gap-4">
        <Skeleton className="h-10 w-10 rounded-full" />
        <SkeletonText lines={2} className="flex-1" />
      </div>
      <SkeletonText lines={3} />
      <div className="flex gap-2">
        <Skeleton className="h-8 w-20 rounded" />
        <Skeleton className="h-8 w-20 rounded" />
      </div>
    </div>
  )
}

function SkeletonTableRow({ className, columns = 4, ...props }: React.ComponentProps<"div"> & { columns?: number }) {
  return (
    <div className={cn("grid gap-4 p-4", `grid-cols-${columns}`, className)} {...props}>
      {Array.from({ length: columns }).map((_, i) => (
        <Skeleton key={i} className="h-4 w-full" />
      ))}
    </div>
  )
}

function SkeletonTable({ className, rows = 5, columns = 4, ...props }: React.ComponentProps<"div"> & { rows?: number; columns?: number }) {
  return (
    <div className={cn("space-y-2", className)} {...props}>
      <SkeletonTableRow columns={columns} className="font-medium" />
      {Array.from({ length: rows }).map((_, i) => (
        <SkeletonTableRow key={i} columns={columns} />
      ))}
    </div>
  )
}

function SkeletonListItem({ className, hasAvatar = true, lines = 2, ...props }: React.ComponentProps<"div"> & { hasAvatar?: boolean; lines?: number }) {
  return (
    <div className={cn("flex items-start gap-3 p-3 rounded-lg hover:bg-accent", className)} {...props}>
      {hasAvatar && <Skeleton className="h-10 w-10 rounded-full shrink-0" />}
      <SkeletonText lines={lines} className="flex-1 min-w-0" />
    </div>
  )
}

function SkeletonList({ className, items = 5, ...props }: React.ComponentProps<"div"> & { items?: number }) {
  return (
    <div className={cn("space-y-2", className)} {...props}>
      {Array.from({ length: items }).map((_, i) => (
        <SkeletonListItem key={i} />
      ))}
    </div>
  )
}

function SkeletonKPI({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div className={cn("rounded-lg border bg-card p-4 space-y-2", className)} {...props}>
      <Skeleton className="h-3 w-3/4" />
      <Skeleton className="h-6 w-1/2" />
    </div>
  )
}

const CHART_BAR_HEIGHTS = [40, 65, 30, 85, 55, 70, 45, 90, 60, 75, 35, 80];

function SkeletonChart({ className, height = 200, ...props }: React.ComponentProps<"div"> & { height?: number }) {
  return (
    <div className={cn("rounded-lg border bg-card p-4", className)} style={{ height }} {...props}>
      <Skeleton className="h-4 w-1/4 mb-4" />
      <div className="h-full flex items-end justify-between gap-1" aria-hidden>
        {Array.from({ length: 12 }).map((_, i) => (
          <Skeleton key={i} className="flex-1 rounded-t" style={{ height: `${CHART_BAR_HEIGHTS[i % CHART_BAR_HEIGHTS.length]}%` }} />
        ))}
      </div>
    </div>
  )
}

function SkeletonEmptyState({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div className={cn("flex flex-col items-center justify-center py-12 text-center", className)} {...props}>
      <Skeleton className="h-12 w-12 rounded-full mb-4" />
      <SkeletonText lines={2} className="w-64 mx-auto" />
    </div>
  )
}

export {
  Skeleton,
  SkeletonText,
  SkeletonCard,
  SkeletonTable,
  SkeletonTableRow,
  SkeletonList,
  SkeletonListItem,
  SkeletonKPI,
  SkeletonChart,
  SkeletonEmptyState,
}