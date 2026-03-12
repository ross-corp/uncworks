import { useState } from "react";
import type { AgentRun, AgentRunPhase } from "../types/agent-run";

function CollapsibleSection({
  label,
  defaultOpen = true,
  onHeaderClick,
  children,
}: {
  label: string;
  defaultOpen?: boolean;
  onHeaderClick?: () => void;
  children?: React.ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const hasChildren = !!children;

  return (
    <div>
      <button
        onClick={() => {
          if (hasChildren) setOpen(!open);
          onHeaderClick?.();
        }}
        className="flex w-full items-center gap-1.5 px-2 py-2 text-xs font-medium uppercase tracking-wider text-txt-tertiary hover:text-txt-secondary transition-colors"
      >
        {hasChildren && (
          <span className="w-3 text-center select-none">
            {open ? "\u25BE" : "\u25B8"}
          </span>
        )}
        <span>{label}</span>
      </button>
      {open && hasChildren && <div className="pb-1">{children}</div>}
    </div>
  );
}

const PHASE_FILTERS: { value: AgentRunPhase | "all"; label: string }[] = [
  { value: "all", label: "All" },
  { value: "running", label: "Running" },
  { value: "waiting_for_input", label: "Waiting" },
  { value: "pending", label: "Pending" },
  { value: "succeeded", label: "Succeeded" },
  { value: "failed", label: "Failed" },
];

export default function Sidebar({
  runs,
  repos,
  selectedRepo,
  onSelectRepo,
  phaseFilter,
  onPhaseFilter,
  onOpenRepos,
  onOpenEvents,
}: {
  runs: AgentRun[];
  repos: { name: string; url: string }[];
  selectedRepo: string | null;
  onSelectRepo: (url: string | null) => void;
  phaseFilter: string;
  onPhaseFilter: (f: string) => void;
  onOpenRepos: () => void;
  onOpenEvents: () => void;
}) {
  function countForPhase(phase: string) {
    let filtered = runs;
    if (selectedRepo) {
      filtered = filtered.filter((r) => r.spec.repoURL === selectedRepo);
    }
    if (phase === "all") return filtered.length;
    return filtered.filter((r) => r.status.phase === phase).length;
  }

  function countForRepo(url: string) {
    return runs.filter((r) => r.spec.repoURL === url).length;
  }

  return (
    <aside className="flex h-screen w-56 flex-col border-r border-edge bg-surface-0">
      <div className="px-4 py-4">
        <h1 className="text-sm font-semibold tracking-tight">
          <span className="text-accent">AOT</span>{" "}
          <span className="text-txt-tertiary font-normal">Agent Orchestration</span>
        </h1>
      </div>

      <div className="flex-1 overflow-y-auto px-2 pb-2 space-y-1">
        {/* AGENT RUNS — phase filters */}
        <CollapsibleSection label="Agent Runs">
          {PHASE_FILTERS.map((s) => {
            const count = countForPhase(s.value);
            return (
              <button
                key={s.value}
                onClick={() => onPhaseFilter(s.value)}
                className={`flex w-full items-center justify-between rounded px-2 py-1.5 pl-6 text-left text-sm transition-colors ${
                  phaseFilter === s.value
                    ? "bg-surface-2 text-txt-primary"
                    : "text-txt-secondary hover:bg-surface-1 hover:text-txt-primary"
                }`}
              >
                <span>{s.label}</span>
                {count > 0 && (
                  <span className="text-xs text-txt-tertiary">{count}</span>
                )}
              </button>
            );
          })}
        </CollapsibleSection>

        {/* REPOS */}
        <CollapsibleSection label="Repositories">
          <button
            onClick={() => onSelectRepo(null)}
            className={`flex w-full items-center justify-between rounded px-2 py-1.5 pl-6 text-left text-sm transition-colors ${
              selectedRepo === null
                ? "bg-surface-2 text-txt-primary"
                : "text-txt-secondary hover:bg-surface-1 hover:text-txt-primary"
            }`}
          >
            <span>All repos</span>
            <span className="text-xs text-txt-tertiary">{runs.length}</span>
          </button>
          {repos.map((repo) => (
            <button
              key={repo.url}
              onClick={() => onSelectRepo(repo.url)}
              className={`flex w-full items-center justify-between rounded px-2 py-1.5 pl-6 text-left text-sm transition-colors ${
                selectedRepo === repo.url
                  ? "bg-surface-2 text-txt-primary"
                  : "text-txt-secondary hover:bg-surface-1 hover:text-txt-primary"
              }`}
            >
              <span className="truncate">{repo.name}</span>
              <span className="text-xs text-txt-tertiary">
                {countForRepo(repo.url)}
              </span>
            </button>
          ))}
        </CollapsibleSection>

        {/* MANAGE REPOS */}
        <CollapsibleSection label="Manage Repos" onHeaderClick={onOpenRepos}>
          {null}
        </CollapsibleSection>

        {/* EVENTS */}
        <CollapsibleSection label="Events" onHeaderClick={onOpenEvents}>
          {null}
        </CollapsibleSection>
      </div>
    </aside>
  );
}
