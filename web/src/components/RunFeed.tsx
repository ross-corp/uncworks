import type { AgentRun } from "../types/agent-run";
import { RunCard } from "./RunCard";
import { Skeleton } from "./Skeleton";
import { Button } from "./ui/button";

function SkeletonCard() {
  return (
    <div className="border-b px-4 py-3" style={{ borderColor: "var(--color-border-subtle)" }}>
      <div className="flex items-start gap-3">
        <Skeleton className="mt-1.5 h-2 w-2 rounded-full" />
        <div className="flex-1 space-y-2">
          <div className="flex items-center justify-between">
            <Skeleton className="h-4 w-32" />
            <Skeleton className="h-3 w-12" />
          </div>
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-3 w-48" />
        </div>
      </div>
    </div>
  );
}

interface RunFeedProps {
  runs: AgentRun[];
  selectedRunId?: string | null;
  onSelectRun?: (run: AgentRun) => void;
  isLoading?: boolean;
  onNewRun?: () => void;
}

export function RunFeed({
  runs,
  selectedRunId,
  onSelectRun,
  isLoading = false,
  onNewRun,
}: RunFeedProps) {
  if (isLoading && runs.length === 0) {
    return (
      <div data-testid="run-feed" className="overflow-y-auto">
        {Array.from({ length: 4 }).map((_, i) => (
          <SkeletonCard key={i} />
        ))}
      </div>
    );
  }

  if (runs.length === 0) {
    return (
      <div data-testid="run-feed" className="flex flex-col items-center justify-center px-6 py-12">
        <p className="text-sm" style={{ color: "var(--color-text-muted)" }}>
          No runs match your filters
        </p>
        {onNewRun && (
          <Button onClick={onNewRun} className="mt-3 text-sm">
            + New Run
          </Button>
        )}
      </div>
    );
  }

  // Sort by creation time descending
  const sorted = [...runs].sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

  return (
    <div data-testid="run-feed" className="overflow-y-auto">
      {sorted.map((run) => (
        <RunCard
          key={run.id}
          run={run}
          selected={selectedRunId === run.id}
          onClick={onSelectRun}
        />
      ))}
    </div>
  );
}
