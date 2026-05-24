// Skeleton primitives for loading states.
// Reuses card/border colors so the shimmer blends with the layout.

import { cn } from "@/lib/utils";

interface SkeletonProps {
  className?: string;
  style?: React.CSSProperties;
}

/** Base shimmer block — sized via className (h-, w-). */
export function Skeleton({ className, style }: SkeletonProps) {
  return (
    <div
      className={cn("animate-pulse rounded-md", className)}
      style={{ background: "var(--surface-2)", ...style }}
    />
  );
}

/** Single-line text skeleton — defaults to a medium-width line. */
export function SkeletonLine({ className, width = "70%" }: { className?: string; width?: string }) {
  return <Skeleton className={cn("h-3", className)} style={{ width }} />;
}

/** Card-shaped placeholder for KPI grids. */
export function SkeletonKPI() {
  return (
    <div
      className="rounded-xl p-5"
      style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
    >
      <SkeletonLine width="50%" />
      <Skeleton className="h-7 mt-2 mb-1" style={{ width: "40%" }} />
      <SkeletonLine width="60%" />
    </div>
  );
}

/** Table-shaped placeholder. */
export function SkeletonTable({ rows = 5, cols = 6 }: { rows?: number; cols?: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="flex gap-3">
          {Array.from({ length: cols }).map((_, c) => (
            <Skeleton key={c} className="h-4 flex-1" />
          ))}
        </div>
      ))}
    </div>
  );
}

/** Generic card-with-list skeleton. */
export function SkeletonList({ count = 3 }: { count?: number }) {
  return (
    <div className="space-y-3">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i}
          className="rounded-xl p-4 space-y-2"
          style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
          <div className="flex gap-2">
            <Skeleton className="h-5 w-20" />
            <Skeleton className="h-5 w-24" />
          </div>
          <SkeletonLine width="85%" />
          <SkeletonLine width="60%" />
        </div>
      ))}
    </div>
  );
}
