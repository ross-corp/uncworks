import { useState, useEffect } from "react";
import { apiFetch } from "../hooks/apiFetch";

interface AutomatedCheck {
  name: string;
  pass: boolean;
  output?: string;
  command?: string;
}

interface CriterionResult {
  scenario: string;
  pass: boolean;
  explanation: string;
}

interface LLMVerdict {
  pass: boolean;
  criteria: CriterionResult[];
  model: string;
}

interface VerificationResult {
  pass: boolean;
  tasksCompleted: number;
  tasksTotal: number;
  validationValid: boolean;
  automatedChecks: AutomatedCheck[];
  llmVerdict?: LLMVerdict;
  failureReport?: string;
  executionTimeMs: number;
}

export default function VerificationPanel({ runId }: { runId: string }) {
  const [result, setResult] = useState<VerificationResult | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    apiFetch(`/api/v1/runs/${runId}/verification`)
      .then((r) => {
        if (!r.ok) return null;
        return r.json();
      })
      .then((data) => {
        if (!cancelled) setResult(data);
      })
      .catch(() => {})
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [runId]);

  if (loading) {
    return (
      <div className="p-4 text-xs text-muted-foreground/60">Loading verification result...</div>
    );
  }

  if (!result) {
    return (
      <div className="p-4 text-xs text-muted-foreground/60">No verification result available</div>
    );
  }

  return (
    <div data-testid="verification-panel" className="p-4 space-y-3 text-xs font-mono">
      {/* Overall verdict */}
      <div className="flex items-center gap-2">
        <span className={`text-sm font-bold ${result.pass ? "text-green-400" : "text-red-400"}`}>
          {result.pass ? "PASSED" : "FAILED"}
        </span>
        <span className="text-muted-foreground">
          in {(result.executionTimeMs / 1000).toFixed(1)}s
        </span>
      </div>

      {/* Gates */}
      <div className="space-y-2">
        <GateRow
          name="Task Completion"
          pass={result.tasksCompleted === result.tasksTotal}
          detail={`${result.tasksCompleted}/${result.tasksTotal} tasks`}
        />
        <GateRow
          name="Spec Validation"
          pass={result.validationValid}
          detail={result.validationValid ? "valid" : "invalid"}
        />
        {result.automatedChecks.map((check, i) => (
          <GateRow
            key={i}
            name={check.name}
            pass={check.pass}
            detail={check.command || ""}
            expandable={!!check.output}
            expandContent={check.output}
          />
        ))}
        {result.llmVerdict && (
          <LLMVerdictSection verdict={result.llmVerdict} />
        )}
      </div>

      {/* Failure report */}
      {result.failureReport && (
        <div className="mt-2 p-2 bg-red-500/10 border border-red-500/30 text-red-300">
          <div className="font-semibold mb-1">Failure Report</div>
          <pre className="whitespace-pre-wrap break-words text-[11px]">{result.failureReport}</pre>
        </div>
      )}
    </div>
  );
}

function GateRow({
  name,
  pass,
  detail,
  expandable,
  expandContent,
}: {
  name: string;
  pass: boolean;
  detail: string;
  expandable?: boolean;
  expandContent?: string;
}) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="border-b border-border/30 pb-1">
      <div className="flex items-center gap-2">
        <span className={pass ? "text-green-400" : "text-red-400"}>
          {pass ? "✓" : "✗"}
        </span>
        <span className="text-foreground">{name}</span>
        <span className="text-muted-foreground/60 ml-auto">{detail}</span>
        {expandable && (
          <button
            onClick={() => setExpanded(!expanded)}
            className="text-muted-foreground hover:text-foreground"
          >
            {expanded ? "▼" : "▶"}
          </button>
        )}
      </div>
      {expanded && expandContent && (
        <pre className="mt-1 p-2 bg-muted/30 text-muted-foreground text-[11px] overflow-x-auto whitespace-pre-wrap break-all max-h-40 overflow-y-auto">
          {expandContent}
        </pre>
      )}
    </div>
  );
}

function LLMVerdictSection({ verdict }: { verdict: LLMVerdict }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="border-b border-border/30 pb-1">
      <div className="flex items-center gap-2">
        <span className={verdict.pass ? "text-green-400" : "text-red-400"}>
          {verdict.pass ? "✓" : "✗"}
        </span>
        <span className="text-foreground">LLM Judge</span>
        <span className="text-muted-foreground/60 ml-auto">{verdict.model}</span>
        <button
          onClick={() => setExpanded(!expanded)}
          className="text-muted-foreground hover:text-foreground"
        >
          {expanded ? "▼" : "▶"}
        </button>
      </div>
      {expanded && (
        <div className="mt-1 ml-5 space-y-1">
          {verdict.criteria.map((c, i) => (
            <div key={i} className="flex gap-2">
              <span className={c.pass ? "text-green-400" : "text-red-400"}>
                {c.pass ? "✓" : "✗"}
              </span>
              <div>
                <div className="text-foreground">{c.scenario}</div>
                <div className="text-muted-foreground/60">{c.explanation}</div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
