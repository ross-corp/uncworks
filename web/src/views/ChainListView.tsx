import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
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
  const navigate = useNavigate();
  const [chains, setChains] = useState<ChainSummary[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    try {
      const resp = await apiFetch("/api/v1/chains");
      if (resp.ok) setChains(await resp.json());
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to load data");
    }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    let cancelled = false;

    const fetch = async () => {
      try {
        const resp = await apiFetch("/api/v1/chains");
        if (resp.ok) {
          const data = await resp.json();
          if (!cancelled) setChains(data);
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

  async function triggerChain(name: string) {
    await apiFetch(`/api/v1/chains/${name}/trigger`, { method: "POST" });
    fetchData();
  }

  async function deleteChain(name: string) {
    const resp = await apiFetch(`/api/v1/chains/${name}`, { method: "DELETE" });
    if (resp.status === 409) {
      const data = await resp.json().catch(() => ({}));
      toast.error(data.error || "Cannot delete chain");
    } else if (resp.ok) {
      toast.success("Chain deleted");
      fetchData();
    } else {
      toast.error("Failed to delete chain");
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <span className="font-semibold flex-1">Chains</span>
        <Button size="sm" variant="outline" onClick={() => navigate("/chains/new")}>
          + new chain
        </Button>
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
            className="flex items-center gap-3 px-4 py-2.5 border-b border-border/40 hover:bg-muted/30 transition-colors"
          >
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-medium">{c.spec.displayName || c.metadata.name}</span>
                <span className="text-xs text-muted-foreground">
                  {c.spec.steps.length} step{c.spec.steps.length !== 1 ? "s" : ""}
                </span>
                {c.spec.projectRef && (
                  <Badge variant="secondary" className="text-xs">{c.spec.projectRef}</Badge>
                )}
              </div>
              {c.spec.description && (
                <div className="text-xs text-muted-foreground mt-0.5">{c.spec.description}</div>
              )}
              <div className="text-xs text-muted-foreground mt-0.5 font-mono">
                {c.spec.steps.map((s) => s.name).join(" -> ")}
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Button
                size="sm"
                variant="ghost"
                onClick={() => triggerChain(c.metadata.name)}
              >
                trigger
              </Button>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => deleteChain(c.metadata.name)}
              >
                delete
              </Button>
              <span className="text-xs text-muted-foreground">{formatAge(c.metadata.creationTimestamp)}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
