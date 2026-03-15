export function Skeleton({ className = "" }: { className?: string }) {
  return (
    <div
      className={`animate-pulse bg-muted ${className}`}
    />
  );
}

export function SkeletonRow() {
  return (
    <tr className="border-b border-border">
      <td className="px-4 py-2.5">
        <Skeleton className="h-4 w-28" />
      </td>
      <td className="px-4 py-2.5">
        <Skeleton className="h-5 w-20" />
      </td>
      <td className="px-4 py-2.5">
        <Skeleton className="h-5 w-14" />
      </td>
      <td className="px-4 py-2.5">
        <Skeleton className="h-5 w-14" />
      </td>
      <td className="px-4 py-2.5">
        <Skeleton className="h-4 w-24" />
      </td>
      <td className="px-4 py-2.5">
        <Skeleton className="h-4 w-40" />
      </td>
      <td className="px-4 py-2.5">
        <Skeleton className="h-4 w-12" />
      </td>
      <td />
    </tr>
  );
}

export function SkeletonDetail() {
  return (
    <div className="flex h-full flex-col border-l border-border bg-background">
      <div className="flex items-center justify-between border-b border-border px-5 py-3">
        <Skeleton className="h-4 w-48" />
      </div>
      <div className="flex-1 p-5 space-y-4">
        <Skeleton className="h-5 w-40" />
        <div className="flex gap-2">
          <Skeleton className="h-5 w-16" />
          <Skeleton className="h-5 w-16" />
          <Skeleton className="h-5 w-16" />
        </div>
        <div className="space-y-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="flex justify-between">
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-3 w-24" />
            </div>
          ))}
        </div>
        <Skeleton className="h-3 w-12" />
        <Skeleton className="h-24 w-full" />
      </div>
    </div>
  );
}
