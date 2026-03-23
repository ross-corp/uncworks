import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { apiFetch } from "../hooks/apiFetch";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";

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

export default function ChainRunDetailView() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [chainRun, setChainRun] = useState<ChainRunDetail | null>(null);
  const [chainDef, setChainDef] = useState<ChainDef | null>(null);

  const fetchData = useCallback(async () => {
    if (!name) return;
    const crResp = await apiFetch(`/api/v1/chainruns/${name}`);
    if (crResp.ok) {
      const cr = await crResp.json();
      setChainRun(cr);
      // Also fetch the chain definition for dependency info
      if (cr.spec?.chainRef) {
        const chainResp = await apiFetch(`/api/v1/chains/${cr.spec.chainRef}`);
        if (chainResp.ok) setChainDef(await chainResp.json());
      }
    }
  }, [name]);

  useEffect(() => {
    fetchData();
    const i = setInterval(fetchData, 5000);
    return () => clearInterval(i);
  }, [fetchData]);

  if (!chainRun) {
    return <div className="flex h-full items-center justify-center text-muted-foreground">Loading...</div>;
  }

  const steps = chainRun.status?.steps || [];
  const chainSteps = chainDef?.spec?.steps || [];

  // Build dependency map for drawing edges
  const depMap = new Map<string, string[]>();
  for (const s of chainSteps) {
    depMap.set(s.name, s.dependsOn || []);
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-4 py-2">
        <div className="flex items-center gap-3">
          <Button size="sm" variant="ghost" className="h-6 text-[11px]" onClick={() => navigate("/")}>
            Runs
          </Button>
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

      {/* DAG visualization */}
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-lg mx-auto space-y-3">
          {steps.map((step) => {
            const deps = depMap.get(step.name) || [];
            const colors = PHASE_COLORS[step.phase] || PHASE_COLORS.pending;

            return (
              <div key={step.name}>
                {/* Dependency arrows */}
                {deps.length > 0 && (
                  <div className="flex items-center justify-center py-1">
                    <span className="text-xs text-muted-foreground">
                      {deps.join(", ")} --&gt;
                    </span>
                  </div>
                )}

                {/* Step node */}
                <div
                  className={`border-2 rounded-lg p-3 cursor-pointer transition-colors ${colors}`}
                  onClick={() => step.runId && navigate(`/run/${step.runId}`)}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-medium text-sm">{step.name}</span>
                    <span className="text-[10px] uppercase tracking-wider">{step.phase}</span>
                  </div>
                  {step.runId && (
                    <span className="text-[10px] text-muted-foreground font-mono">{step.runId}</span>
                  )}
                  {step.message && step.message !== "completed" && step.message !== "started" && (
                    <div className="text-xs text-muted-foreground mt-1 truncate">{step.message}</div>
                  )}
                </div>
              </div>
            );
          })}
        </div>

        {chainRun.status?.message && (
          <div className="mt-4 text-center text-xs text-muted-foreground">{chainRun.status.message}</div>
        )}
      </div>
    </div>
  );
}
