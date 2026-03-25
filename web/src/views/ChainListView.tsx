import { useState, useEffect, useCallback } from "react";
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

export default function ChainListView() {
  const [chains, setChains] = useState<ChainSummary[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    try {
      const resp = await apiFetch("/api/v1/chains");
      if (resp.ok) setChains(await resp.json());
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
        <span className="font-semibold">Chains</span>
      </div>

      <div className="flex-1 overflow-y-auto">
        {loading && (
          <div className="flex h-full items-center justify-center text-muted-foreground">Loading...</div>
        )}

        {!loading && chains.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground">
            No chains defined
          </div>
        )}

        {!loading && chains.map((c) => (
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
              <Button
                size="sm"
                variant="ghost"
                className="h-6 text-[11px] px-2"
                onClick={() => triggerChain(c.metadata.name)}
              >
                trigger
              </Button>
              <span className="text-[11px] text-muted-foreground">{formatAge(c.metadata.creationTimestamp)}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
