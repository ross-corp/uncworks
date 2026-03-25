import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from "./ui/collapsible";
import { Button } from "./ui/button";
import type { AgentRun } from "../types/agent-run";

interface FailureDiagnosisPanelProps {
  run: AgentRun;
  elapsed: string;
  onViewTraces: () => void;
  onRetry: () => void;
  onEditRetry: () => void;
  onArchive: () => void;
}

export default function FailureDiagnosisPanel({
  run,
  elapsed,
  onViewTraces,
  onRetry,
  onEditRetry,
  onArchive,
}: FailureDiagnosisPanelProps) {
  const [open, setOpen] = useState(true);

  const stage = run.status.stage || "unknown stage";
  const errorMessage = run.status.message || "An unexpected error occurred.";

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <div className="border-l-4 border-red-500 bg-red-500/5 mx-4 my-2 rounded-r">
        <CollapsibleTrigger asChild>
          <button className="flex items-center gap-2 w-full px-3 py-2 text-left hover:bg-red-500/10 transition-colors">
            {open ? (
              <ChevronDown className="size-3.5 text-red-400 shrink-0" />
            ) : (
              <ChevronRight className="size-3.5 text-red-400 shrink-0" />
            )}
            <span className="text-sm font-semibold text-red-500">
              Failed during {stage}
            </span>
            {elapsed && (
              <span className="ml-auto text-xs text-muted-foreground">{elapsed}</span>
            )}
          </button>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="px-3 pb-3 space-y-3">
            <p className="text-xs text-red-400/90 font-mono break-words whitespace-pre-wrap">
              {errorMessage}
            </p>
            <div className="flex flex-wrap gap-2">
              <Button size="sm" variant="outline" className="text-xs" onClick={onViewTraces}>
                View in Traces
              </Button>
              <Button size="sm" variant="outline" className="text-xs" onClick={onRetry}>
                Retry
              </Button>
              <Button size="sm" variant="outline" className="text-xs" onClick={onEditRetry}>
                Edit &amp; Retry
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="text-xs text-muted-foreground"
                onClick={onArchive}
              >
                Archive
              </Button>
            </div>
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}
