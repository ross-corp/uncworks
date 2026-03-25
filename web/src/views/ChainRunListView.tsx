import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { apiFetch } from "../hooks/apiFetch";
import { formatAge } from "../lib/format";
import { Badge } from "../components/ui/badge";

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

  const fetchData = useCallback(async () => {
    try {
      const resp = await apiFetch("/api/v1/chainruns");
      if (resp.ok) setChainRuns(await resp.json());
    } catch { /* silent */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    fetchData();
    const i = setInterval(fetchData, 10000);
    return () => clearInterval(i);
  }, [fetchData]);

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b px-4 py-2">
        <span className="font-semibold">Chain Runs</span>
      </div>

      <div className="flex-1 overflow-y-auto">
        {loading && (
          <div className="flex h-full items-center justify-center text-muted-foreground">Loading...</div>
        )}

        {!loading && chainRuns.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground">
            No chain runs yet
          </div>
        )}

        {!loading && chainRuns.map((cr) => (
          <div
            key={cr.metadata.name}
            className="flex items-center gap-3 px-4 py-3 border-b border-border/50 cursor-pointer hover:bg-muted/30 transition-colors"
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
              <div className="text-[10px] text-muted-foreground mt-0.5">
                {cr.spec.triggeredBy && `triggered by ${cr.spec.triggeredBy}`}
              </div>
            </div>
            <span className="text-[11px] text-muted-foreground shrink-0">{formatAge(cr.metadata.creationTimestamp)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
