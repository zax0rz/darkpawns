export function Skeleton({ className = '' }: { className?: string }) {
  return <div className={`animate-pulse bg-paper rounded ${className}`} />;
}

export function TableSkeleton({ rows = 5, cols = 4 }: { rows?: number; cols?: number }) {
  return (
    <div className="bg-paper-deep rounded-none border border-rule overflow-hidden">
      {/* Header */}
      <div className="border-b border-rule px-4 py-3 flex gap-4">
        {Array.from({ length: cols }).map((_, i) => (
          <Skeleton key={i} className="h-3 flex-1" />
        ))}
      </div>
      {/* Rows */}
      {Array.from({ length: rows }).map((_, rowIdx) => (
        <div
          key={rowIdx}
          className="border-b border-rule px-4 py-3 flex gap-4"
        >
          {Array.from({ length: cols }).map((_, colIdx) => (
            <Skeleton key={colIdx} className="h-4 flex-1" />
          ))}
        </div>
      ))}
    </div>
  );
}

export function CardSkeleton() {
  return (
    <div className="bg-paper-deep rounded-none border border-rule p-5">
      <Skeleton className="h-5 w-32 mb-3" />
      <div className="space-y-2">
        <Skeleton className="h-4 w-48" />
        <Skeleton className="h-4 w-40" />
        <Skeleton className="h-4 w-36" />
      </div>
    </div>
  );
}

export function StatCardSkeleton() {
  return (
    <div className="bg-paper-deep rounded-none border border-rule p-4">
      <Skeleton className="h-3 w-16 mb-2" />
      <Skeleton className="h-8 w-20" />
    </div>
  );
}
