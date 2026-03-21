import { useState, useEffect, useCallback, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import type { AgentRun, AgentRunPhase } from "../types/agent-run";
import { useClient, mapRun } from "../hooks/useClient";
import { useToast } from "../components/Toast";
import RunStatusBadge from "../components/RunStatusBadge";

/** Compute aggregate status for a group of runs. */
function aggregatePhase(runs: AgentRun[]): AgentRunPhase {
  if (runs.some((r) => r.status.phase === "succeeded")) return "succeeded";
  if (runs.some((r) => r.status.phase === "running")) return "running";
  if (runs.some((r) => r.status.phase === "waiting_for_input")) return "waiting_for_input";
  if (runs.some((r) => r.status.phase === "pending")) return "pending";
  if (runs.every((r) => r.status.phase === "failed")) return "failed";
  if (runs.every((r) => r.status.phase === "cancelled")) return "cancelled";
  return "pending";
}

function formatAge(iso: string): string {
  if (!iso) return "";
  const secs = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (secs < 60) return `${secs}s`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h`;
  return `${Math.floor(hrs / 24)}d`;
}

export default function FeatureDetailView() {
  const { name } = useParams<{ name: string }>();
  const client = useClient();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [allRuns, setAllRuns] = useState<AgentRun[]>([]);
  const [selected, setSelected] = useState(0);
  const [loading, setLoading] = useState(true);

  const featureName = name ? decodeURIComponent(name) : "";

  const fetchRuns = useCallback(async () => {
    try {
      const result = await client.listAgentRuns();
      setAllRuns(result.map(mapRun));
    } catch {
      // silent
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
              toast("Retry run created", "success");
              navigate(`/run/${run.id}`);
            })
            .catch(() => {
              toast("Failed to create retry run", "error");
            });
          break;
        }
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [featureRuns, selected, navigate, client, featureName, toast]);

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

      {/* Table header */}
      <div className="grid grid-cols-[1fr_100px_80px_100px_70px] gap-2 border-b px-4 py-1 text-xs text-muted-foreground uppercase tracking-wider">
        <span>Name</span>
        <span>Status</span>
        <span>Stage</span>
        <span>Model</span>
        <span>Age</span>
      </div>

      {/* Run rows */}
      <div className="flex-1 overflow-y-auto">
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
            className={`grid grid-cols-[1fr_100px_80px_100px_70px] gap-2 px-4 py-2 text-sm cursor-pointer transition-colors ${
              index === selected ? "bg-accent text-accent-foreground" : "hover:bg-muted/50"
            }`}
            onClick={() => navigate(`/run/${run.id}`)}
          >
            <span className="truncate">{run.spec.displayName || run.name}</span>
            <RunStatusBadge phase={run.status.phase} />
            <span className="text-muted-foreground text-xs">{run.status.stage || ""}</span>
            <span className="text-muted-foreground text-xs truncate">{run.spec.modelTier || ""}</span>
            <span className="text-muted-foreground text-xs">{formatAge(run.createdAt)}</span>
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
