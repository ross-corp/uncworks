import { useState, useEffect } from "react";
import type { AgentRun } from "../types/agent-run";
import { PhaseBadge, BackendBadge, ModelTierBadge } from "./StatusBadge";

export default function AgentRunDetailPanel({
  run,
  onClose,
  onCancel,
  onSendInput,
}: {
  run: AgentRun;
  onClose: () => void;
  onCancel: (id: string) => void;
  onSendInput: (id: string, input: string) => void;
}) {
  const [humanInput, setHumanInput] = useState("");

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  function repoName(url: string): string {
    const parts = url.split("/");
    return parts.slice(-2).join("/");
  }

  function formatTime(iso: string): string {
    if (!iso) return "\u2014";
    return new Date(iso).toLocaleString();
  }

  function duration(): string {
    if (!run.status.startedAt) return "\u2014";
    const start = new Date(run.status.startedAt).getTime();
    const end = run.status.completedAt
      ? new Date(run.status.completedAt).getTime()
      : Date.now();
    const secs = Math.floor((end - start) / 1000);
    if (secs < 60) return `${secs}s`;
    const mins = Math.floor(secs / 60);
    const remSecs = secs % 60;
    if (mins < 60) return `${mins}m ${remSecs}s`;
    return `${Math.floor(mins / 60)}h ${mins % 60}m`;
  }

  const isActive = run.status.phase === "running" || run.status.phase === "waiting_for_input";
  const isWaiting = run.status.phase === "waiting_for_input";

  return (
    <div className="flex h-full flex-col border-l border-edge bg-surface-0">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-edge px-5 py-3">
        <span className="font-mono text-sm text-txt-tertiary truncate">{run.id}</span>
        <button onClick={onClose} className="btn-ghost px-2 text-lg">
          &times;
        </button>
      </div>

      {/* Body */}
      <div className="flex-1 overflow-y-auto p-5">
        {/* Name & badges */}
        <h2 className="mb-2 text-base font-semibold">{run.name}</h2>
        <div className="mb-4 flex flex-wrap items-center gap-2">
          <PhaseBadge phase={run.status.phase} />
          <BackendBadge backend={run.spec.backend} />
          <ModelTierBadge tier={run.spec.modelTier} />
        </div>

        {/* Metadata grid */}
        <div className="mb-4 space-y-2">
          <MetaRow label="Repository" value={repoName(run.spec.repoURL)} mono />
          <MetaRow label="Branch" value={run.spec.branch} mono />
          <MetaRow label="Created" value={formatTime(run.createdAt)} />
          <MetaRow label="Started" value={formatTime(run.status.startedAt)} />
          <MetaRow label="Duration" value={duration()} />
          {run.status.podName && (
            <MetaRow label="Pod" value={run.status.podName} mono />
          )}
          {run.status.traceID && (
            <MetaRow label="Trace ID" value={run.status.traceID} mono />
          )}
          <MetaRow label="TTL" value={`${run.spec.ttlSeconds}s`} />
        </div>

        {/* Env vars */}
        {Object.keys(run.spec.envVars).length > 0 && (
          <div className="mb-4">
            <h3 className="mb-1 text-xs font-medium uppercase tracking-wider text-txt-tertiary">
              Environment
            </h3>
            <div className="space-y-1">
              {Object.entries(run.spec.envVars).map(([k, v]) => (
                <div key={k} className="font-mono text-xs">
                  <span className="text-txt-tertiary">{k}</span>
                  <span className="text-txt-secondary">=</span>
                  <span className="text-txt-primary">{v}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Prompt */}
        <div className="mb-4">
          <h3 className="mb-1 text-xs font-medium uppercase tracking-wider text-txt-tertiary">
            Prompt
          </h3>
          <div className="rounded border border-edge bg-surface-2 p-3 text-sm text-txt-secondary whitespace-pre-wrap">
            {run.spec.prompt}
          </div>
        </div>

        {/* Status message */}
        {run.status.message && (
          <div className="mb-4">
            <h3 className="mb-1 text-xs font-medium uppercase tracking-wider text-txt-tertiary">
              Status Message
            </h3>
            <div
              className={`rounded border p-3 text-sm whitespace-pre-wrap ${
                run.status.phase === "failed"
                  ? "border-red-500/30 bg-red-500/5 text-red-400"
                  : run.status.phase === "waiting_for_input"
                  ? "border-purple-500/30 bg-purple-500/5 text-purple-400"
                  : "border-edge bg-surface-2 text-txt-secondary"
              }`}
            >
              {run.status.message}
            </div>
          </div>
        )}

        {/* HITL Input */}
        {isWaiting && (
          <div className="mb-4">
            <h3 className="mb-1 text-xs font-medium uppercase tracking-wider text-txt-tertiary">
              Human Input
            </h3>
            <textarea
              className="input-field min-h-[80px] resize-y"
              value={humanInput}
              onChange={(e) => setHumanInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                  e.preventDefault();
                  if (humanInput.trim()) {
                    onSendInput(run.id, humanInput.trim());
                    setHumanInput("");
                  }
                }
              }}
              placeholder="Type your response to the agent..."
            />
            <button
              onClick={() => {
                if (humanInput.trim()) {
                  onSendInput(run.id, humanInput.trim());
                  setHumanInput("");
                }
              }}
              disabled={!humanInput.trim()}
              className="btn-primary mt-2 text-sm disabled:opacity-40"
            >
              Send Input
            </button>
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="flex justify-between border-t border-edge px-5 py-3">
        <div className="flex items-center gap-2">
          {run.status.traceID && (
            <button
              onClick={() => navigator.clipboard.writeText(run.status.traceID)}
              className="btn-ghost text-sm"
            >
              Copy Trace
            </button>
          )}
        </div>
        {isActive && (
          <button
            onClick={() => onCancel(run.id)}
            className="rounded bg-danger px-3 py-1.5 text-sm font-medium text-white hover:bg-danger/80 transition-colors"
          >
            Cancel Run
          </button>
        )}
      </div>
    </div>
  );
}

function MetaRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-4">
      <span className="text-xs text-txt-tertiary whitespace-nowrap">{label}</span>
      <span
        className={`text-sm text-txt-secondary text-right truncate ${mono ? "font-mono text-xs" : ""}`}
      >
        {value}
      </span>
    </div>
  );
}
