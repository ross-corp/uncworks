import { useState, useEffect, useCallback, useMemo } from "react";
import { usePoll } from "../hooks/usePoll";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import type { AgentRun, AgentRunPhase } from "../types/agent-run";
import { useClient, mapRun } from "../hooks/useClient";
import { apiFetch } from "../hooks/apiFetch";
import { formatAge, formatDuration, aggregatePhase } from "../lib/format";
import RunStatusBadge from "../components/RunStatusBadge";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import {
  Empty,
  EmptyHeader,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
} from "../components/ui/empty";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel,
  AlertDialogContent, AlertDialogDescription, AlertDialogFooter,
  AlertDialogHeader, AlertDialogTitle,
} from "../components/ui/alert-dialog";
import CustomSelect from "../components/CustomSelect";

// ---- Chain run types ----
interface ChainRunSummary {
  metadata: { name: string; creationTimestamp: string };
  spec: { chainRef: string; triggeredBy?: string };
  status: { phase?: string; startedAt?: string; completedAt?: string };
}

// ---- Unified run type ----
type RunKind = "agent" | "chain" | "scheduled";

interface UnifiedRun {
  id: string;
  kind: RunKind;
  name: string;
  status: string;
  createdAt: string;
  // agent-only
  agentRun?: AgentRun;
  // chain-only
  chainRun?: ChainRunSummary;
}

type ViewMode = "features" | "all" | "unified";
type FilterField = "name" | "state" | "stage" | "model" | null;

const FILTER_KEYS: Record<string, { field: FilterField; label: string; placeholder: string }> = {
  "/": { field: "name", label: "Name", placeholder: "filter by name..." },
  "?": { field: "state", label: "State", placeholder: "filter by state (running, failed, succeeded)..." },
  "'": { field: "stage", label: "Stage", placeholder: "filter by stage (plan, execute, verify)..." },
  '"': { field: "model", label: "Model", placeholder: "filter by model..." },
};

const FIELD_OPTIONS: { field: FilterField; label: string }[] = [
  { field: "name", label: "Name" },
  { field: "state", label: "State" },
  { field: "stage", label: "Stage" },
  { field: "model", label: "Model" },
];



interface FeatureGroup {
  feature: string;
  runs: AgentRun[];
  phase: AgentRunPhase;
  prUrl?: string;
}

function ExternalStatus({ run }: { run: AgentRun }) {
  const vrPass = (() => {
    if (!run.status.verificationResult) return null;
    try {
      const vr = JSON.parse(run.status.verificationResult) as { pass: boolean };
      return vr.pass;
    } catch {
      return null;
    }
  })();

  const awaitingApproval = run.status.phase === "waiting_for_input" &&
    (!run.spec.approvalMode || run.spec.approvalMode === "hitl" || run.spec.approvalMode === "hybrid");

  if (!run.status.prUrl && !run.status.lastCIStatus && vrPass === null && !awaitingApproval) return null;
  return (
    <div className="flex items-center gap-1 shrink-0">
      {awaitingApproval && (
        <span
          className="text-xs font-medium bg-amber-500/15 text-amber-600 dark:text-amber-400 px-1.5 py-0.5 rounded-md animate-pulse"
          title="Waiting for human approval"
        >
          ⏳ Approve
        </span>
      )}
      {vrPass !== null && (
        <span
          className={`text-xs font-medium px-1.5 py-0.5 rounded-md ${
            vrPass
              ? "bg-green-500/15 text-green-600 dark:text-green-400"
              : "bg-red-500/15 text-red-600 dark:text-red-400"
          }`}
          title={vrPass ? "Verification passed" : "Verification failed"}
        >
          V{vrPass ? "✓" : "✗"}
        </span>
      )}
      {run.status.prUrl && (
        <a
          href={run.status.prUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs font-medium bg-blue-500/15 text-blue-500 hover:bg-blue-500/25 px-1.5 py-0.5 rounded-md transition-colors"
          onClick={(e) => e.stopPropagation()}
        >
          PR
        </a>
      )}
      {run.status.lastCIStatus === "success" && (
        <span className="text-xs font-medium bg-green-500/15 text-green-600 dark:text-green-400 px-1.5 py-0.5 rounded-md">
          CI ✓
        </span>
      )}
      {run.status.lastCIStatus === "failure" && (
        <span className="text-xs font-medium bg-red-500/15 text-red-600 dark:text-red-400 px-1.5 py-0.5 rounded-md">
          CI ✗
        </span>
      )}
    </div>
  );
}

function FeatureHeader({
  group,
  expanded,
  onToggle,
  onNavigate,
}: {
  group: FeatureGroup;
  expanded: boolean;
  onToggle: () => void;
  onNavigate: () => void;
}) {
  return (
    <div
      className="flex items-center gap-3 border-b bg-muted/20 px-4 py-2 text-sm cursor-pointer select-none hover:bg-muted/40 transition-colors"
      onClick={onToggle}
    >
      <span className="text-sm text-foreground transition-transform" style={{ transform: expanded ? "rotate(90deg)" : "none" }}>&#9654;</span>
      <span
        className="font-semibold truncate hover:underline"
        onClick={(e) => { e.stopPropagation(); onNavigate(); }}
      >
        {group.feature}
      </span>
      <RunStatusBadge phase={group.phase} />
      <span className="text-xs text-muted-foreground">
        {group.runs.length} run{group.runs.length !== 1 ? "s" : ""}
      </span>
      {group.prUrl && (
        <a
          href={group.prUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-blue-500 hover:underline ml-auto"
          onClick={(e) => e.stopPropagation()}
        >
          PR
        </a>
      )}
    </div>
  );
}

export default function RunListView() {
  const client = useClient();
  const navigate = useNavigate();
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [chainRuns, setChainRuns] = useState<ChainRunSummary[]>([]);
  const [selected, setSelected] = useState(0);
  const [filter, setFilter] = useState("");
  const [filterField, setFilterField] = useState<FilterField>(null);
  const [loading, setLoading] = useState(true);
  const [activeProject, setActiveProject] = useState("");
  const [viewMode, setViewMode] = useState<ViewMode>("unified");
  const [projectPickerOpen, setProjectPickerOpen] = useState(false);
  const [collapsedFeatures, setCollapsedFeatures] = useState<Set<string>>(new Set());
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [showArchived, setShowArchived] = useState(false);
  const [selectMode, setSelectMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [pendingDeleteRun, setPendingDeleteRun] = useState<AgentRun | null>(null);

  const fetchRuns = useCallback(async () => {
    try {
      const result = await client.listAgentRuns();
      setRuns(result.map(mapRun));
    } catch {
      toast.error("Failed to load runs");
    } finally {
      setLoading(false);
    }
  }, [client]);

  usePoll(async () => {
    try {
      const [agentResult, chainResp] = await Promise.allSettled([
        client.listAgentRuns(),
        apiFetch("/api/v1/chainruns"),
      ]);
      if (agentResult.status === "fulfilled") {
        setRuns(agentResult.value.map(mapRun));
      } else {
        toast.error("Failed to load agent runs");
      }
      if (chainResp.status === "fulfilled") {
        if (chainResp.value.ok) {
          const data = await chainResp.value.json();
          setChainRuns(Array.isArray(data) ? data : []);
        } else {
          console.error("Failed to fetch chain runs:", chainResp.value.status, chainResp.value.statusText);
        }
      } else {
        console.error("Chain runs fetch error:", chainResp.reason);
      }
    } catch {
      toast.error("Failed to load runs");
    } finally {
      setLoading(false);
    }
  }, 5000, [client]);

  const projects = useMemo(() => {
    const set = new Set<string>();
    for (const r of runs) {
      if (r.spec.project) set.add(r.spec.project);
    }
    return Array.from(set).sort();
  }, [runs]);

  const projectFiltered = useMemo(() => {
    if (!activeProject) return runs;
    return runs.filter((r) => r.spec.project === activeProject);
  }, [runs, activeProject]);

  const filtered = useMemo(() => {
    return projectFiltered.filter((r) => {
      if (!showArchived && r.status.archived) return false;
      if (statusFilter === "running" && r.status.phase !== "running" && r.status.phase !== "waiting_for_input" && r.status.phase !== "pending") return false;
      if (statusFilter === "failed" && r.status.phase !== "failed") return false;
      if (statusFilter === "succeeded" && r.status.phase !== "succeeded") return false;
      if (statusFilter === "waiting" && r.status.phase !== "waiting_for_input") return false;
      if (!filter) return true;
      const q = filter.toLowerCase();
      switch (filterField) {
        case "name": return (r.spec.displayName || r.name).toLowerCase().includes(q);
        case "state": return r.status.phase.toLowerCase().includes(q);
        case "stage": return (r.status.stage || "").toLowerCase().includes(q);
        case "model": return (r.spec.modelTier || "").toLowerCase().includes(q);
        default:
          return (
            (r.spec.displayName || r.name).toLowerCase().includes(q) ||
            r.status.phase.toLowerCase().includes(q) ||
            (r.spec.modelTier || "").toLowerCase().includes(q) ||
            (r.spec.feature || "").toLowerCase().includes(q)
          );
      }
    });
  }, [projectFiltered, filter, filterField, statusFilter, showArchived]);

  const featureGroups = useMemo((): FeatureGroup[] => {
    const map = new Map<string, AgentRun[]>();
    for (const r of filtered) {
      const key = r.spec.feature || "";
      const arr = map.get(key);
      if (arr) arr.push(r); else map.set(key, [r]);
    }
    const groups: FeatureGroup[] = [];
    const named = Array.from(map.entries()).filter(([k]) => k !== "").sort(([a], [b]) => a.localeCompare(b));
    for (const [feature, groupRuns] of named) {
      groups.push({ feature, runs: groupRuns, phase: aggregatePhase(groupRuns), prUrl: groupRuns.find((r) => r.status.prUrl)?.status.prUrl });
    }
    const unassigned = map.get("");
    if (unassigned) {
      groups.push({ feature: "Unassigned", runs: unassigned, phase: aggregatePhase(unassigned), prUrl: unassigned.find((r) => r.status.prUrl)?.status.prUrl });
    }
    return groups;
  }, [filtered]);

  // Agent-run list used by features/all view modes and as the keyboard nav source for those modes.
  const visibleRuns = useMemo((): AgentRun[] => {
    if (viewMode === "all") return filtered;
    if (viewMode === "unified") return filtered;
    const result: AgentRun[] = [];
    for (const group of featureGroups) {
      if (!collapsedFeatures.has(group.feature)) result.push(...group.runs);
    }
    return result;
  }, [viewMode, filtered, featureGroups, collapsedFeatures]);

  // Unified view: merge agent + chain runs sorted newest first.
  const unifiedRuns = useMemo((): UnifiedRun[] => {
    const agentEntries: UnifiedRun[] = filtered.map((r) => {
      const isScheduled = !!(r.spec as { scheduleName?: string }).scheduleName;
      return {
        id: r.id,
        kind: isScheduled ? "scheduled" : "agent",
        name: r.spec.displayName || r.name,
        status: r.status.phase,
        createdAt: r.createdAt,
        agentRun: r,
      };
    });

    const chainEntries: UnifiedRun[] = chainRuns
      .filter((cr) => {
        if (!filter) return true;
        const q = filter.toLowerCase();
        return cr.metadata.name.toLowerCase().includes(q) ||
          (cr.spec.chainRef || "").toLowerCase().includes(q) ||
          (cr.status.phase || "").toLowerCase().includes(q);
      })
      .filter((cr) => {
        if (statusFilter === "all") return true;
        const phase = cr.status.phase || "pending";
        if (statusFilter === "running") return phase === "running";
        if (statusFilter === "failed") return phase === "failed";
        if (statusFilter === "succeeded") return phase === "succeeded";
        return true;
      })
      .map((cr) => ({
        id: cr.metadata.name,
        kind: (cr.spec.triggeredBy ? "scheduled" : "chain") as RunKind,
        name: cr.metadata.name,
        status: cr.status.phase || "pending",
        createdAt: cr.metadata.creationTimestamp,
        chainRun: cr,
      }));

    return [...agentEntries, ...chainEntries].sort(
      (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    );
  }, [filtered, chainRuns, filter, statusFilter]);

  // The list keyboard nav actually operates on, depending on active view mode.
  // In unified mode j/k/Enter/d/c index into unifiedRuns (agent + chain).
  // In features/all modes they index into visibleRuns (agent-only).
  const navList = viewMode === "unified" ? unifiedRuns : visibleRuns;

  const toggleFeature = useCallback((feature: string) => {
    setCollapsedFeatures((prev) => {
      const next = new Set(prev);
      if (next.has(feature)) next.delete(feature); else next.add(feature);
      return next;
    });
  }, []);

  function toggleSelect(id: string) {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  }

  async function archiveSelected() {
    if (selectedIds.size === 0) return;
    const count = selectedIds.size;
    toast.loading(`Archiving ${count} runs...`, { id: "archive" });
    try {
      await apiFetch("/api/v1/runs/bulk-archive", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ runIds: Array.from(selectedIds), archived: true }),
      });
      toast.success(`${count} runs archived`, { id: "archive" });
      setSelectedIds(new Set());
      setSelectMode(false);
      fetchRuns();
    } catch {
      toast.error("Archive failed — try again", { id: "archive" });
    }
  }

  const hasActiveFilter = statusFilter !== "all" || activeProject !== "" || filter !== "";

  function clearAllFilters() {
    setStatusFilter("all");
    setActiveProject("");
    setFilter("");
    setFilterField(null);
  }

  // Build an empty-state message that distinguishes "no runs exist" from "filter matches nothing".
  function emptyStateMessage(listEmpty: boolean): { title: string; description?: string; showCTA?: boolean } {
    if (!listEmpty) return { title: "" };
    if (filter) return { title: "No runs match filter", description: "Try adjusting your filters to see more results." };
    if (statusFilter !== "all") return { title: `No ${statusFilter} runs`, description: "Try changing the status filter to see all runs." };
    return { 
      title: "No runs yet", 
      description: "Submit your first run with `uncworks run --repo <url> --prompt <text>` or click the button below.",
      showCTA: true
    };
  }

  function EmptyStateContent() {
    const message = emptyStateMessage(true);
    if (!message.title) return null;
    
    return (
      <Empty className="h-full border-0">
        <EmptyHeader>
          <EmptyTitle>{message.title}</EmptyTitle>
          {message.description && (
            <EmptyDescription className="font-mono text-xs">
              {message.description}
            </EmptyDescription>
          )}
        </EmptyHeader>
        {message.showCTA && (
          <EmptyContent>
            <Button onClick={() => navigate("/new")}>
              Create First Run
            </Button>
          </EmptyContent>
        )}
      </Empty>
    );
  }

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      // Don't fire shortcuts when the user is typing in a focused input/select/textarea.
      const tag = (e.target as HTMLElement).tagName;
      if (tag === "INPUT" || tag === "SELECT" || tag === "TEXTAREA") return;

      if (projectPickerOpen) {
        if (e.key === "Escape") setProjectPickerOpen(false);
        return;
      }
      if (filterField !== null) {
        if (e.key === "Escape") { setFilter(""); setFilterField(null); }
        return;
      }
      const filterDef = FILTER_KEYS[e.key];
      if (filterDef) { e.preventDefault(); setFilterField(filterDef.field); return; }

      switch (e.key) {
        case "j": setSelected((s) => Math.min(s + 1, navList.length - 1)); break;
        case "k": setSelected((s) => Math.max(s - 1, 0)); break;
        case "Enter": {
          const cur = navList[selected];
          if (!cur) break;
          if ("agentRun" in cur && cur.agentRun) navigate(`/run/${cur.id}`);
          else if ("chainRun" in cur && cur.chainRun) navigate(`/chainrun/${cur.id}`);
          else navigate(`/run/${cur.id}`);
          break;
        }
        case "n": navigate("/new"); break;
        case "p": setProjectPickerOpen(true); break;
        case "x": setSelectMode((prev) => !prev); setSelectedIds(new Set()); break;
        case "a":
          if (selectMode) { setShowArchived((prev) => !prev); }
          break;
        case "1": setViewMode("features"); setSelected(0); break;
        case "2": setViewMode("all"); setSelected(0); break;
        case "3": setViewMode("unified"); setSelected(0); break;
        case "d": {
          // d only deletes agent runs; chain runs have no delete endpoint.
          const cur = navList[selected];
          if (!cur) break;
          const agentRun = "agentRun" in cur ? cur.agentRun : undefined;
          if (agentRun) {
            setPendingDeleteRun(agentRun);
          } else if (!("agentRun" in cur)) {
            // non-unified view: navList is AgentRun[]
            setPendingDeleteRun(cur as unknown as AgentRun);
          }
          break;
        }
        case "c": {
          // c clones the focused agent run; silently no-ops for chain runs.
          const cur = navList[selected];
          if (!cur) break;
          const agentRun = "agentRun" in cur ? cur.agentRun : undefined;
          if (agentRun) {
            navigate(`/new?clone=${agentRun.id}`);
          } else if (!("agentRun" in cur)) {
            // non-unified view: navList is AgentRun[]
            navigate(`/new?clone=${(cur as unknown as AgentRun).id}`);
          }
          break;
        }
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [navList, selected, filterField, projectPickerOpen, selectMode, showArchived, navigate]);

  // Clamp selection when the list shrinks.
  useEffect(() => {
    if (selected >= navList.length) setSelected(Math.max(0, navList.length - 1));
  }, [navList.length, selected]);

  function RunRow({ run, index }: { run: AgentRun; index: number }) {
    const hasDiff = !!(run.status.totalAdditions || run.status.totalDeletions);
    return (
      <div
        data-testid={`run-row-${run.id}`}
        className={`flex items-center gap-3 px-4 py-2.5 cursor-pointer border-b border-border/40 transition-colors ${
          index === selected ? "bg-accent/40 outline outline-1 outline-border/60" : "hover:bg-muted/30"
        } ${run.status.archived ? "opacity-40" : ""}`}
        role="button"
        tabIndex={0}
        onClick={() => selectMode ? toggleSelect(run.id) : navigate(`/run/${run.id}`)}
        onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); selectMode ? toggleSelect(run.id) : navigate(`/run/${run.id}`); } }}
      >
        {selectMode && (
          <input
            type="checkbox"
            checked={selectedIds.has(run.id)}
            onChange={() => toggleSelect(run.id)}
            className="shrink-0 rounded"
          />
        )}

        <RunStatusBadge phase={run.status.phase} stage={run.status.stage} />

        <div className="flex-1 min-w-0">
          <span className="truncate block text-sm">{run.spec.displayName || run.name}</span>
        </div>

        <ExternalStatus run={run} />

        <div className="flex items-center gap-2.5 shrink-0">
          {run.status.totalCost && (
            <span className="text-xs text-muted-foreground">{run.status.totalCost}</span>
          )}

          {hasDiff && (
            <span className="text-xs font-mono">
              <span className="text-green-600 dark:text-green-400">+{run.status.totalAdditions || 0}</span>
              <span className="text-red-600 dark:text-red-400 ml-1">-{run.status.totalDeletions || 0}</span>
            </span>
          )}

          {run.status.phase === "succeeded" && !hasDiff && (
            <span
              className="text-xs text-amber-500/70 font-mono"
              title={run.status.message || "Succeeded with no code changes"}
            >
              ∅
            </span>
          )}

          <span className="text-xs text-muted-foreground bg-muted/60 px-2 py-0.5 rounded-md">
            {run.spec.modelTier || "default"}
          </span>

          {run.status.completedAt && run.status.startedAt && (
            <span className="text-xs text-muted-foreground font-mono w-14 text-right shrink-0">
              {formatDuration(run.status.startedAt, run.status.completedAt)}
            </span>
          )}

          <span className="text-xs text-muted-foreground w-10 text-right">{formatAge(run.createdAt)}</span>
        </div>
      </div>
    );
  }

  function getVisibleIndex(run: AgentRun): number {
    return visibleRuns.indexOf(run);
  }

  const KIND_BADGE: Record<RunKind, { label: string; className: string }> = {
    agent: { label: "one-shot", className: "bg-blue-500/15 text-blue-500" },
    chain: { label: "chain", className: "bg-purple-500/15 text-purple-500" },
    scheduled: { label: "scheduled", className: "bg-amber-500/15 text-amber-600 dark:text-amber-400" },
  };

  function UnifiedRunRow({ ur, index }: { ur: UnifiedRun; index: number }) {
    const badge = KIND_BADGE[ur.kind];
    return (
      <div
        data-testid={`run-row-${ur.id}`}
        className={`flex items-center gap-3 px-4 py-2.5 cursor-pointer border-b border-border/40 transition-colors ${
          index === selected ? "bg-accent/40 outline outline-1 outline-border/60" : "hover:bg-muted/30"
        }`}
        onClick={() => {
          if (ur.agentRun) navigate(`/run/${ur.id}`);
          else navigate(`/chainrun/${ur.id}`);
        }}
      >
        <span className={`text-xs font-medium px-1.5 py-0.5 rounded-md shrink-0 ${badge.className}`}>
          {badge.label}
        </span>

        <RunStatusBadge phase={ur.status} />

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 min-w-0">
            <span className="truncate text-sm">{ur.name}</span>
            {ur.chainRun && (
              <span className="text-xs text-muted-foreground shrink-0">chain: {ur.chainRun.spec.chainRef}</span>
            )}
            {ur.chainRun?.spec.triggeredBy && (
              <span className="text-xs text-muted-foreground shrink-0">via {ur.chainRun.spec.triggeredBy}</span>
            )}
          </div>
        </div>

        {ur.agentRun && <ExternalStatus run={ur.agentRun} />}

        <span className="text-xs text-muted-foreground w-10 text-right shrink-0">{formatAge(ur.createdAt)}</span>
      </div>
    );
  }

  const activeFilterDef = filterField
    ? Object.values(FILTER_KEYS).find((d) => d.field === filterField)
    : null;

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="border-b px-4 space-y-2 pb-2">
        <div className="h-12 flex items-center gap-2">
          <div className="flex items-center gap-3 flex-1">
            <span className="font-semibold text-base">Runs</span>
            <span className="text-muted-foreground text-xs">
              {viewMode === "unified" ? unifiedRuns.length : filtered.length}
            </span>
            {activeProject && (
              <Badge variant="secondary" className="cursor-pointer" onClick={() => setActiveProject("")}>
                {activeProject} &times;
              </Badge>
            )}
            <div className="flex items-center gap-0.5 bg-muted/50 rounded-md p-0.5 ml-2">
              {(["unified", "features", "all"] as const).map((m) => (
                <button
                  key={m}
                  onClick={() => setViewMode(m)}
                  className={`px-2 py-0.5 text-xs rounded-md transition-colors ${
                    viewMode === m
                      ? "bg-background text-foreground font-medium shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {m}
                </button>
              ))}
            </div>
          </div>

        </div>

        {/* Filter row */}
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-0.5 bg-muted/50 rounded-md p-0.5">
            {(["all", "running", "waiting", "failed", "succeeded"] as const).map((s) => (
              <button
                key={s}
                onClick={() => setStatusFilter(s)}
                className={`px-2.5 py-1 text-xs rounded-md transition-colors ${
                  statusFilter === s
                    ? "bg-background text-foreground font-medium shadow-sm"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {s}
              </button>
            ))}
          </div>
          {hasActiveFilter && (
            <button
              onClick={clearAllFilters}
              className="text-xs px-2 py-1 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
            >
              clear &times;
            </button>
          )}
          <div className="flex-1" />
          <button
            onClick={() => setShowArchived(!showArchived)}
            className={`text-xs px-2.5 py-1 rounded-md transition-colors ${showArchived ? "bg-muted text-foreground" : "text-muted-foreground hover:text-foreground"}`}
          >
            {showArchived ? "hide archived" : "show archived"}
          </button>
          <button
            onClick={() => { setSelectMode(!selectMode); setSelectedIds(new Set()); }}
            className={`text-xs px-2.5 py-1 rounded-md transition-colors ${selectMode ? "bg-muted text-foreground" : "text-muted-foreground hover:text-foreground"}`}
          >
            {selectMode ? "done" : "select"}
          </button>
        </div>

        {/* Persistent filter bar */}
        <div className="flex items-center gap-2">
          <CustomSelect
            value={filterField ?? ""}
            onChange={(v) => setFilterField((v as FilterField) || null)}
            options={[{ value: "", label: "Field..." }, ...FIELD_OPTIONS.map(({ field, label }) => ({ value: field ?? "", label }))]}
            className="text-xs"
          />
          <div className="relative flex-1 flex items-center">
            {filterField && (
              <span className="absolute left-2 text-xs font-medium text-blue-500 pointer-events-none">
                {activeFilterDef?.label}:
              </span>
            )}
            <input
              className={`w-full bg-muted/30 border border-border/50 rounded-md text-sm outline-none px-2 py-1 focus:border-border transition-colors ${filterField ? "pl-14" : ""}`}
              placeholder={filterField ? (activeFilterDef?.placeholder ?? "filter...") : "filter runs..."}
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              onFocus={() => { if (!filterField) setFilterField("name"); }}
              onKeyDown={(e) => { if (e.key === "Escape") { setFilter(""); setFilterField(null); (e.target as HTMLInputElement).blur(); } }}
            />
            {filter && (
              <button
                className="absolute right-2 text-muted-foreground hover:text-foreground text-xs"
                onClick={() => { setFilter(""); setFilterField(null); }}
              >
                &times;
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Project picker */}
      {projectPickerOpen && (
        <div className="border-b bg-background px-4 py-2">
          <div className="text-xs text-muted-foreground mb-1">Select project (esc to close):</div>
          <div
            className={`cursor-pointer px-2 py-1 text-sm rounded ${!activeProject ? "bg-accent text-accent-foreground" : "hover:bg-muted/50"}`}
            onClick={() => { setActiveProject(""); setProjectPickerOpen(false); setSelected(0); }}
          >
            all projects
          </div>
          {projects.map((p) => (
            <div
              key={p}
              className={`cursor-pointer px-2 py-1 text-sm rounded ${activeProject === p ? "bg-accent text-accent-foreground" : "hover:bg-muted/50"}`}
              onClick={() => { setActiveProject(p); setProjectPickerOpen(false); setSelected(0); }}
            >
              {p}
            </div>
          ))}
        </div>
      )}

      {/* Mass select action bar */}
      {selectMode && selectedIds.size > 0 && (
        <div className="flex items-center gap-3 border-b bg-blue-500/10 px-4 py-1.5">
          <span className="text-xs font-medium">{selectedIds.size} selected</span>
          <Button size="sm" variant="destructive" onClick={archiveSelected}>
            Archive
          </Button>
          <Button size="sm" variant="ghost" onClick={() => { setSelectedIds(new Set()); setSelectMode(false); }}>
            Cancel
          </Button>
        </div>
      )}

      {/* Run rows */}
      <div className="flex-1 overflow-y-auto overscroll-none">
        {loading && unifiedRuns.length === 0 && filtered.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground">Loading...</div>
        )}
        {!loading && viewMode === "unified" && unifiedRuns.length === 0 && (
          <EmptyStateContent />
        )}
        {!loading && viewMode !== "unified" && filtered.length === 0 && (
          <EmptyStateContent />
        )}

        {viewMode === "unified"
          ? unifiedRuns.map((ur, i) => <UnifiedRunRow key={`${ur.kind}-${ur.id}`} ur={ur} index={i} />)
          : viewMode === "features"
          ? featureGroups.map((group) => (
              <div key={group.feature}>
                <FeatureHeader
                  group={group}
                  expanded={!collapsedFeatures.has(group.feature)}
                  onToggle={() => toggleFeature(group.feature)}
                  onNavigate={() => {
                    if (group.feature && group.feature !== "Unassigned") {
                      navigate(`/feature/${encodeURIComponent(group.feature)}`);
                    }
                  }}
                />
                {!collapsedFeatures.has(group.feature) &&
                  group.runs.map((run) => (
                    <RunRow key={run.id} run={run} index={getVisibleIndex(run)} />
                  ))}
              </div>
            ))
          : filtered.map((run, i) => <RunRow key={run.id} run={run} index={i} />)}
      </div>

      {/* Footer */}
      <div className="border-t px-4 py-1.5 text-xs text-muted-foreground flex items-center justify-between">
        <span>j/k nav &middot; enter open &middot; n new &middot; d delete &middot; c clone &middot; x select &middot; p project</span>
        <span>/ name &middot; ? state &middot; &apos; stage &middot; &quot; model</span>
      </div>

      <AlertDialog open={!!pendingDeleteRun} onOpenChange={(open) => { if (!open) setPendingDeleteRun(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete run?</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingDeleteRun?.spec.displayName || pendingDeleteRun?.name} will be permanently deleted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={() => {
              if (pendingDeleteRun) {
                apiFetch(`/api/v1/runs/${pendingDeleteRun.id}`, { method: "DELETE" })
                  .then(() => fetchRuns())
                  .catch(() => toast.error("Delete failed — try again"));
                setPendingDeleteRun(null);
              }
            }}>Delete</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
