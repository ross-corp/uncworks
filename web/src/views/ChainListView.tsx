import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { apiFetch } from "../hooks/apiFetch";
import { formatAge } from "../lib/format";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";

interface ChainSummary {
  metadata: { name: string; creationTimestamp: string };
  spec: {
    displayName?: string;
    description?: string;
    projectRef?: string;
    steps: { name: string; templateRef: string; dependsOn?: string[] }[];
  };
}

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

export default function ChainListView() {
  const navigate = useNavigate();
  const [chains, setChains] = useState<ChainSummary[]>([]);
  const [chainRuns, setChainRuns] = useState<ChainRunSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<"chains" | "runs">("chains");

  const fetchData = useCallback(async () => {
    try {
      const [chainsResp, runsResp] = await Promise.all([
        apiFetch("/api/v1/chains"),
        apiFetch("/api/v1/chainruns"),
      ]);
      if (chainsResp.ok) setChains(await chainsResp.json());
      if (runsResp.ok) setChainRuns(await runsResp.json());
    } catch { /* silent */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    fetchData();
    const i = setInterval(fetchData, 10000);
    return () => clearInterval(i);
  }, [fetchData]);

  async function triggerChain(name: string) {
    await apiFetch(`/api/v1/chains/${name}/trigger`, { method: "POST" });
    fetchData();
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b px-4 py-2">
        <div className="flex items-center gap-3">
          <span className="font-semibold">Chains</span>
          <div className="flex gap-0.5 border rounded px-0.5 py-0.5">
            {(["chains", "runs"] as const).map((t) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={`px-2 py-0.5 text-[11px] rounded transition-colors ${
                  tab === t
                    ? "bg-foreground text-background font-medium"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {t === "chains" ? `Chains (${chains.length})` : `Runs (${chainRuns.length})`}
              </button>
            ))}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="ghost" className="h-6 text-[11px]" onClick={() => navigate("/")}>
            Runs
          </Button>
          <Button size="sm" variant="ghost" className="h-6 text-[11px]" onClick={() => navigate("/schedules")}>
            Schedules
          </Button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {loading && (
          <div className="flex h-full items-center justify-center text-muted-foreground">Loading...</div>
        )}

        {tab === "chains" && !loading && (
          <>
            {chains.length === 0 && (
              <div className="flex h-full items-center justify-center text-muted-foreground">
                No chains defined
              </div>
            )}
            {chains.map((c) => (
              <div
                key={c.metadata.name}
                className="flex items-center gap-3 px-4 py-3 border-b border-border/50 hover:bg-muted/30 transition-colors"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{c.spec.displayName || c.metadata.name}</span>
                    <span className="text-[11px] text-muted-foreground">
                      {c.spec.steps.length} step{c.spec.steps.length !== 1 ? "s" : ""}
                    </span>
                    {c.spec.projectRef && (
                      <Badge variant="secondary" className="text-[10px]">{c.spec.projectRef}</Badge>
                    )}
                  </div>
                  {c.spec.description && (
                    <div className="text-xs text-muted-foreground mt-0.5">{c.spec.description}</div>
                  )}
                  <div className="text-[10px] text-muted-foreground mt-0.5 font-mono">
                    {c.spec.steps.map((s) => s.name).join(" -> ")}
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <button
                    onClick={() => triggerChain(c.metadata.name)}
                    className="text-[11px] px-2 py-0.5 rounded border text-muted-foreground hover:text-foreground transition-colors"
                  >
                    trigger
                  </button>
                  <span className="text-[11px] text-muted-foreground">{formatAge(c.metadata.creationTimestamp)}</span>
                </div>
              </div>
            ))}
          </>
        )}

        {tab === "runs" && !loading && (
          <>
            {chainRuns.length === 0 && (
              <div className="flex h-full items-center justify-center text-muted-foreground">
                No chain runs yet
              </div>
            )}
            {chainRuns.map((cr) => (
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
          </>
        )}
      </div>
    </div>
  );
}
