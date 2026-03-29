import { useState, useEffect, useCallback, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import type { AgentRun } from "../types/agent-run";
import { useClient, mapRun } from "../hooks/useClient";
import { toast } from "sonner";
import { formatAge, aggregatePhase } from "../lib/format";
import RunStatusBadge from "../components/RunStatusBadge";

export default function FeatureDetailView() {
  const { name } = useParams<{ name: string }>();
  const client = useClient();
  const navigate = useNavigate();
  const [allRuns, setAllRuns] = useState<AgentRun[]>([]);
  const [selected, setSelected] = useState(0);
  const [loading, setLoading] = useState(true);

  const featureName = name ? decodeURIComponent(name) : "";

  const fetchRuns = useCallback(async () => {
    try {
      const result = await client.listAgentRuns();
      setAllRuns(result.map(mapRun));
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to load runs");
    } finally {
      setLoading(false);
    }
  }, [client]);

  // Poll every 5s
  useEffect(() => {
    fetchRuns();
    const interval = setInterval(fetchRuns, 5000);
    return () => clearInterval(interval);
  }, [fetchRuns]);

  // Filter runs for this feature
  const featureRuns = useMemo(() => {
    return allRuns.filter((r) => r.spec.feature === featureName);
  }, [allRuns, featureName]);

  const aggPhase = useMemo(() => aggregatePhase(featureRuns), [featureRuns]);

  const prUrl = useMemo(() => {
    return featureRuns.find((r) => r.status.prUrl)?.status.prUrl;
  }, [featureRuns]);

  // Keep selection in bounds
  useEffect(() => {
    if (selected >= featureRuns.length) setSelected(Math.max(0, featureRuns.length - 1));
  }, [featureRuns.length, selected]);

  // Keyboard navigation
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      const tag = (e.target as HTMLElement).tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
      switch (e.key) {
        case "Escape":
          navigate("/");
          break;
        case "j":
          setSelected((s) => Math.min(s + 1, featureRuns.length - 1));
          break;
        case "k":
          setSelected((s) => Math.max(s - 1, 0));
          break;
        case "Enter":
          if (featureRuns[selected]) navigate(`/run/${featureRuns[selected].id}`);
          break;
        case "r": {
          e.preventDefault();
          const lastRun = featureRuns[0]; // most recent
          if (!lastRun) break;

          let retryPrompt = lastRun.spec.prompt || "";
          if (lastRun.status.phase === "failed" && lastRun.status.message) {
            retryPrompt = `PREVIOUS ATTEMPT FAILED: ${lastRun.status.message}\n\n${retryPrompt}`;
          }

          client
            .createAgentRun({
              backend: "pod",
              repos: lastRun.spec.repos,
              prompt: retryPrompt,
              ttlSeconds: lastRun.spec.ttlSeconds || 900,
              modelTier: lastRun.spec.modelTier || "default",
              feature: featureName,
              ...(lastRun.spec.project ? { project: lastRun.spec.project } : {}),
              ...(lastRun.spec.tags?.length ? { tags: lastRun.spec.tags } : {}),
              ...(lastRun.spec.specContent
                ? { specContent: lastRun.spec.specContent, orchestrationMode: "spec-driven" as const }
                : {}),
            })
            .then((run) => {
              toast.success("Retry run created");
              navigate(`/run/${run.id}`);
            })
            .catch(() => {
              toast.error("Failed to create retry run");
            });
          break;
        }
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [featureRuns, selected, navigate, client, featureName]);

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-4 py-2">
        <div className="flex items-center gap-3">
          <span className="font-semibold truncate">{featureName}</span>
          <RunStatusBadge phase={aggPhase} />
          <span className="text-xs text-muted-foreground">
            {featureRuns.length} run{featureRuns.length !== 1 ? "s" : ""}
          </span>
          {prUrl && (
            <a
              href={prUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-blue-500 hover:underline"
            >
              PR
            </a>
          )}
        </div>
        <span className="text-xs text-muted-foreground">esc back · r retry · enter detail</span>
      </div>

      {/* Run rows */}
      <div className="flex-1 overflow-y-auto overscroll-none">
        {loading && featureRuns.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground">Loading...</div>
        )}
        {!loading && featureRuns.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground">
            No runs for feature &quot;{featureName}&quot;
          </div>
        )}

        {featureRuns.map((run, index) => (
          <div
            key={run.id}
            className={`flex items-center gap-3 px-4 py-2 text-sm cursor-pointer border-b border-border/50 transition-colors ${
              index === selected ? "bg-accent/50" : "hover:bg-muted/30"
            }`}
            onClick={() => navigate(`/run/${run.id}`)}
          >
            <RunStatusBadge phase={run.status.phase} stage={run.status.stage} />
            <div className="flex-1 min-w-0">
              <span className="truncate block">{run.spec.displayName || run.name}</span>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              {run.status.stage && (
                <span className="text-xs text-muted-foreground">{run.status.stage}</span>
              )}
              <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                {run.spec.modelTier || "default"}
              </span>
              <span className="text-xs text-muted-foreground w-8 text-right">{formatAge(run.createdAt)}</span>
            </div>
          </div>
        ))}
      </div>

      {/* Footer shortcuts */}
      <div className="border-t px-4 py-1 text-xs text-muted-foreground">
        j/k navigate · enter detail · r retry · esc back
      </div>
    </div>
  );
}
