import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { apiFetch } from "../hooks/apiFetch";
import { formatAge } from "../lib/format";
import { Badge } from "../components/ui/badge";
import { Spinner } from "../components/ui/spinner";
import {
  Empty,
  EmptyHeader,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
} from "../components/ui/empty";
import { Button } from "../components/ui/button";

interface ChainRunSummary {
  metadata: { name: string; creationTimestamp: string };
  spec: { chainRef: string; triggeredBy?: string };
  status: {
    phase?: string;
    startedAt?: string;
    completedAt?: string;
  };
}

const PHASE_COLORS: Record<string, string> = {
  succeeded: "border-green-500/40 text-green-500",
  failed: "text-red-500",
  running: "text-blue-500 animate-pulse",
  pending: "text-muted-foreground",
};

export default function ChainRunListView() {
  const navigate = useNavigate();
  const [chainRuns, setChainRuns] = useState<ChainRunSummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    const fetch = async () => {
      try {
        const resp = await apiFetch("/api/v1/chainruns");
        if (resp.ok) {
          const data = await resp.json();
          if (!cancelled) setChainRuns(data);
        }
      } catch (e) {
        if (!cancelled) toast.error(e instanceof Error ? e.message : "Failed to load data");
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    fetch();
    const i = setInterval(() => {
      if (!cancelled) fetch();
    }, 10000);

    return () => {
      cancelled = true;
      clearInterval(i);
    };
  }, []);

  return (
    <div className="flex h-full flex-col">
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <span className="font-semibold">Chain Runs</span>
      </div>

      <div className="flex-1 overflow-y-auto">
        {loading && (
          <div className="flex h-full items-center justify-center">
            <Spinner className="text-muted-foreground" />
          </div>
        )}

        {!loading && chainRuns.length === 0 && (
          <Empty className="h-full border-0">
            <EmptyHeader>
              <EmptyTitle>No chain runs yet</EmptyTitle>
              <EmptyDescription>Chain runs appear here when a chain is triggered manually or by a schedule.</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button size="sm" variant="ghost" onClick={() => navigate("/chains")}>View Chains</Button>
            </EmptyContent>
          </Empty>
        )}

        {!loading && chainRuns.map((cr) => (
          <div
            key={cr.metadata.name}
            className="flex items-center gap-3 px-4 py-2.5 border-b border-border/40 cursor-pointer hover:bg-muted/30 transition-colors"
            onClick={() => navigate(`/chainrun/${cr.metadata.name}`)}
          >
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-medium font-mono text-sm">{cr.metadata.name}</span>
                <span className="text-xs text-muted-foreground">chain: {cr.spec.chainRef}</span>
                {cr.status.phase && (
                  <Badge
                    variant={cr.status.phase === "succeeded" ? "outline" : cr.status.phase === "failed" ? "destructive" : "default"}
                    className={PHASE_COLORS[cr.status.phase] || ""}
                  >
                    {cr.status.phase}
                  </Badge>
                )}
              </div>
              <div className="text-xs text-muted-foreground mt-0.5">
                {cr.spec.triggeredBy && `triggered by ${cr.spec.triggeredBy}`}
              </div>
            </div>
            <span className="text-xs text-muted-foreground shrink-0">{formatAge(cr.metadata.creationTimestamp)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
