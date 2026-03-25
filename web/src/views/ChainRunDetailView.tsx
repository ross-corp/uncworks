import { useState, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { apiFetch } from "../hooks/apiFetch";
import { Badge } from "../components/ui/badge";
import ErrorBoundary from "../components/ErrorBoundary";
import ChainDagViz, { type ChainStepStatus } from "../components/ChainDagViz";

interface StepStatus {
  name: string;
  phase: string;
  runId?: string;
  startedAt?: string;
  completedAt?: string;
  message?: string;
}

interface ChainRunDetail {
  metadata: { name: string; creationTimestamp: string };
  spec: { chainRef: string; triggeredBy?: string };
  status: {
    phase?: string;
    steps?: StepStatus[];
    startedAt?: string;
    completedAt?: string;
    message?: string;
  };
}

interface ChainDef {
  spec: {
    steps: { name: string; templateRef: string; dependsOn?: string[] }[];
  };
}

const PHASE_COLORS: Record<string, string> = {
  succeeded: "border-green-500 bg-green-500/10 text-green-700 dark:text-green-400",
  failed: "border-red-500 bg-red-500/10 text-red-700 dark:text-red-400",
  running: "border-blue-500 bg-blue-500/10 text-blue-700 dark:text-blue-400 animate-pulse",
  pending: "border-muted bg-muted/30 text-muted-foreground",
  skipped: "border-muted bg-muted/10 text-muted-foreground/50",
};

const PHASE_BADGE: Record<string, string> = {
  succeeded: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  failed: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  running: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  pending: "bg-muted text-muted-foreground",
  skipped: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
};

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

function elapsedSecs(startedAt?: string, completedAt?: string): number | undefined {
  if (!startedAt) return undefined;
  const end = completedAt ? new Date(completedAt) : new Date();
  const start = new Date(startedAt);
  return Math.max(0, Math.round((end.getTime() - start.getTime()) / 1000));
}

type Tab = "dag" | "timeline";

export default function ChainRunDetailView() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [chainRun, setChainRun] = useState<ChainRunDetail | null>(null);
  const [chainDef, setChainDef] = useState<ChainDef | null>(null);
  const [tab, setTab] = useState<Tab>("dag");

  useEffect(() => {
    let cancelled = false;

    const fetch = async () => {
      if (!name) return;
      const crResp = await apiFetch(`/api/v1/chainruns/${name}`);
      if (crResp.ok) {
        const cr = await crResp.json();
        if (!cancelled) setChainRun(cr);
        if (cr.spec?.chainRef) {
          const chainResp = await apiFetch(`/api/v1/chains/${cr.spec.chainRef}`);
          if (chainResp.ok) {
            const chainData = await chainResp.json();
            if (!cancelled) setChainDef(chainData);
          }
        }
      }
    };

    fetch();
    const i = setInterval(() => {
      if (!cancelled) fetch();
    }, 5000);

    return () => {
      cancelled = true;
      clearInterval(i);
    };
  }, [name]);

  if (!chainRun) {
    return <div className="flex h-full items-center justify-center text-muted-foreground">Loading...</div>;
  }

  const steps = chainRun.status?.steps || [];
  const chainSteps = chainDef?.spec?.steps || [];

  // Build dependency map
  const depMap = new Map<string, string[]>();
  for (const s of chainSteps) {
    depMap.set(s.name, s.dependsOn || []);
  }

  // Build ChainStepStatus array for DAG viz
  const dagSteps: ChainStepStatus[] = steps.map((s) => ({
    name: s.name,
    phase: s.phase,
    dependsOn: depMap.get(s.name) || [],
    runId: s.runId,
    elapsedSeconds: elapsedSecs(s.startedAt, s.completedAt),
  }));

  // Text-based fallback step list
  const stepFallback = (
    <div className="max-w-lg mx-auto space-y-3">
      {steps.map((step) => {
        const deps = depMap.get(step.name) || [];
        const colors = PHASE_COLORS[step.phase] || PHASE_COLORS.pending;
        return (
          <div key={step.name}>
            {deps.length > 0 && (
              <div className="flex items-center justify-center py-1">
                <span className="text-xs text-muted-foreground">
                  {deps.join(", ")} --&gt;
                </span>
              </div>
            )}
            <div
              className={`border-2 rounded-lg p-3 cursor-pointer transition-colors ${colors}`}
              onClick={() => step.runId && navigate(`/run/${step.runId}`)}
            >
              <div className="flex items-center justify-between">
                <span className="font-medium text-sm">{step.name}</span>
                <span className="text-xs uppercase tracking-wider">{step.phase}</span>
              </div>
              {step.runId && (
                <span className="text-xs text-muted-foreground font-mono">{step.runId}</span>
              )}
              {step.message && step.message !== "completed" && step.message !== "started" && (
                <div className="text-xs text-muted-foreground mt-1 truncate">{step.message}</div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <div className="flex flex-col gap-0.5">
          <div className="text-xs text-muted-foreground">
            <Link to="/chains" className="hover:text-foreground transition-colors">Chains</Link>
            {" / "}
            <span>{chainRun.metadata.name}</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="font-semibold">Chain: {chainRun.spec.chainRef}</span>
            {chainRun.status?.phase && (
              <Badge variant={chainRun.status.phase === "succeeded" ? "outline" : chainRun.status.phase === "failed" ? "destructive" : "default"}>
                {chainRun.status.phase}
              </Badge>
            )}
            {chainRun.spec.triggeredBy && (
              <span className="text-xs text-muted-foreground">{chainRun.spec.triggeredBy}</span>
            )}
          </div>
        </div>
      </div>

      {/* Tab switcher */}
      <div className="flex gap-1 border-b px-4 pt-2">
        {(["dag", "timeline"] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-3 py-1.5 text-sm font-medium rounded-t transition-colors capitalize ${
              tab === t
                ? "border-b-2 border-primary text-foreground"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {t === "dag" ? "DAG" : "Timeline"}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {tab === "dag" && (
          <div className="h-full">
            {dagSteps.length === 0 ? (
              <div className="flex h-full items-center justify-center text-muted-foreground text-sm">
                No steps yet.
              </div>
            ) : (
              <ErrorBoundary fallback={<div className="p-6 overflow-y-auto">{stepFallback}</div>}>
                <ChainDagViz steps={dagSteps} />
              </ErrorBoundary>
            )}
          </div>
        )}

        {tab === "timeline" && (
          <div className="overflow-y-auto p-6 h-full">
            <div className="max-w-2xl mx-auto space-y-2">
              {steps.length === 0 && (
                <div className="text-center text-muted-foreground text-sm">No steps yet.</div>
              )}
              {steps.map((step, idx) => {
                const secs = elapsedSecs(step.startedAt, step.completedAt);
                const badgeClass = PHASE_BADGE[step.phase] || PHASE_BADGE.pending;
                // Compute bar width relative to max duration
                return (
                  <div
                    key={step.name}
                    className="flex items-center gap-3 cursor-pointer group"
                    onClick={() => step.runId && navigate(`/run/${step.runId}`)}
                  >
                    <span className="w-5 text-xs text-muted-foreground text-right shrink-0">{idx + 1}</span>
                    <span className="w-32 text-sm font-medium truncate shrink-0 group-hover:text-primary transition-colors">
                      {step.name}
                    </span>
                    <span className={`px-2 py-0.5 rounded text-xs uppercase font-semibold shrink-0 ${badgeClass}`}>
                      {step.phase}
                    </span>
                    {step.startedAt && (
                      <span className="text-xs text-muted-foreground shrink-0">
                        {new Date(step.startedAt).toLocaleTimeString()}
                      </span>
                    )}
                    {secs !== undefined && (
                      <span className="text-xs text-muted-foreground shrink-0">
                        {formatDuration(secs)}
                      </span>
                    )}
                    {step.message && step.message !== "completed" && step.message !== "started" && (
                      <span className="text-xs text-muted-foreground truncate">{step.message}</span>
                    )}
                  </div>
                );
              })}
            </div>
            {chainRun.status?.message && (
              <div className="mt-4 text-center text-xs text-muted-foreground">{chainRun.status.message}</div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
