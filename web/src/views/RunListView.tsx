import { useState, useEffect, useCallback, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import type { AgentRun, AgentRunPhase } from "../types/agent-run";
import { useClient, mapRun } from "../hooks/useClient";
import { apiFetch } from "../hooks/apiFetch";
import RunStatusBadge from "../components/RunStatusBadge";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";

type ViewMode = "features" | "all";
type FilterField = "name" | "state" | "stage" | "model" | null;

const FILTER_KEYS: Record<string, { field: FilterField; label: string; placeholder: string }> = {
  "/": { field: "name", label: "/", placeholder: "filter by name..." },
  "?": { field: "state", label: "?", placeholder: "filter by state (running, failed, succeeded)..." },
  "'": { field: "stage", label: "'", placeholder: "filter by stage (plan, execute, verify)..." },
  '"': { field: "model", label: '"', placeholder: "filter by model..." },
};

function aggregatePhase(runs: AgentRun[]): AgentRunPhase {
  if (runs.some((r) => r.status.phase === "succeeded")) return "succeeded";
  if (runs.some((r) => r.status.phase === "running")) return "running";
  if (runs.some((r) => r.status.phase === "waiting_for_input")) return "waiting_for_input";
  if (runs.some((r) => r.status.phase === "pending")) return "pending";
  if (runs.every((r) => r.status.phase === "failed")) return "failed";
  if (runs.every((r) => r.status.phase === "cancelled")) return "cancelled";
  return "pending";
}

interface FeatureGroup {
  feature: string;
  runs: AgentRun[];
  phase: AgentRunPhase;
  prUrl?: string;
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
      className="flex items-center gap-3 border-b bg-muted/30 px-4 py-2 text-sm cursor-pointer select-none"
      onClick={onToggle}
    >
      <span className="text-xs text-muted-foreground">{expanded ? "\u25BC" : "\u25B6"}</span>
      <span
        className="font-bold truncate hover:underline"
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
  const [selected, setSelected] = useState(0);
  const [filter, setFilter] = useState("");
  const [filterField, setFilterField] = useState<FilterField>(null);
  const [loading, setLoading] = useState(true);
  const [activeProject, setActiveProject] = useState("");
  const [viewMode, setViewMode] = useState<ViewMode>("features");
  const [projectPickerOpen, setProjectPickerOpen] = useState(false);
  const [collapsedFeatures, setCollapsedFeatures] = useState<Set<string>>(new Set());
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [showArchived, setShowArchived] = useState(false);
  const [selectMode, setSelectMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const fetchRuns = useCallback(async () => {
    try {
      const result = await client.listAgentRuns();
      setRuns(result.map(mapRun));
    } catch {
      // silent
    } finally {
      setLoading(false);
    }
  }, [client]);

  useEffect(() => {
    fetchRuns();
    const interval = setInterval(fetchRuns, 5000);
    return () => clearInterval(interval);
  }, [fetchRuns]);

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

  const visibleRuns = useMemo((): AgentRun[] => {
    if (viewMode === "all") return filtered;
    const result: AgentRun[] = [];
    for (const group of featureGroups) {
      if (!collapsedFeatures.has(group.feature)) result.push(...group.runs);
    }
    return result;
  }, [viewMode, filtered, featureGroups, collapsedFeatures]);

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
    if (!window.confirm(`Archive ${selectedIds.size} run(s)?`)) return;
    await apiFetch("/api/v1/runs/bulk-archive", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ runIds: Array.from(selectedIds), archived: true }),
    });
    setSelectedIds(new Set());
    setSelectMode(false);
    fetchRuns();
  }

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
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
        case "j": setSelected((s) => Math.min(s + 1, visibleRuns.length - 1)); break;
        case "k": setSelected((s) => Math.max(s - 1, 0)); break;
        case "Enter": if (visibleRuns[selected]) navigate(`/run/${visibleRuns[selected].id}`); break;
        case "n": navigate("/new"); break;
        case "p": setProjectPickerOpen(true); break;
        case "x": setSelectMode(!selectMode); setSelectedIds(new Set()); break;
        case "a":
          if (selectMode) { setShowArchived(!showArchived); }
          break;
        case "1": setViewMode("features"); setSelected(0); break;
        case "2": setViewMode("all"); setSelected(0); break;
        case "d":
          if (visibleRuns[selected]) {
            const run = visibleRuns[selected];
            if (window.confirm(`Delete ${run.spec.displayName || run.name}?`)) {
              apiFetch(`/api/v1/runs/${run.id}`, { method: "DELETE" }).then(() => fetchRuns());
            }
          }
          break;
        case "c":
          if (visibleRuns[selected]) navigate(`/new?clone=${visibleRuns[selected].id}`);
          break;
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [visibleRuns, selected, filterField, projectPickerOpen, selectMode, showArchived, navigate, fetchRuns]);

  useEffect(() => {
    if (selected >= visibleRuns.length) setSelected(Math.max(0, visibleRuns.length - 1));
  }, [visibleRuns.length, selected]);

  function RunRow({ run, index }: { run: AgentRun; index: number }) {
    const hasDiff = !!(run.status.totalAdditions || run.status.totalDeletions);
    return (
      <div
        data-testid={`run-row-${run.id}`}
        className={`flex items-center gap-3 px-4 py-2 text-sm cursor-pointer border-b border-border/50 transition-colors ${
          index === selected ? "bg-accent/50" : "hover:bg-muted/30"
        } ${run.status.archived ? "opacity-40" : ""}`}
        onClick={() => selectMode ? toggleSelect(run.id) : navigate(`/run/${run.id}`)}
      >
        {/* Checkbox (select mode) */}
        {selectMode && (
          <input
            type="checkbox"
            checked={selectedIds.has(run.id)}
            onChange={() => toggleSelect(run.id)}
            className="shrink-0"
          />
        )}

        {/* Status dot */}
        <RunStatusBadge phase={run.status.phase} />

        {/* Name + inline metadata */}
        <div className="flex-1 min-w-0">
          <span className="truncate block">{run.spec.displayName || run.name}</span>
        </div>

        {/* Inline metadata pills */}
        <div className="flex items-center gap-2 shrink-0">
          {/* Model */}
          <span className="text-[11px] text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
            {run.spec.modelTier || "default"}
          </span>

          {/* Cost */}
          {run.status.totalCost && (
            <span className="text-[11px] text-muted-foreground">{run.status.totalCost}</span>
          )}

          {/* Diff stats */}
          {hasDiff && (
            <span className="text-[11px] font-mono">
              <span className="text-green-600 dark:text-green-400">+{run.status.totalAdditions || 0}</span>
              <span className="text-red-600 dark:text-red-400"> -{run.status.totalDeletions || 0}</span>
            </span>
          )}

          {/* PR link */}
          {run.status.prUrl && (
            <a
              href={run.status.prUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-[11px] text-blue-500 hover:text-blue-400 font-medium"
              onClick={(e) => e.stopPropagation()}
            >
              PR
            </a>
          )}

          {/* Age */}
          <span className="text-[11px] text-muted-foreground w-8 text-right">{formatAge(run.createdAt)}</span>
        </div>
      </div>
    );
  }

  function getVisibleIndex(run: AgentRun): number {
    return visibleRuns.indexOf(run);
  }

  const activeFilterDef = filterField
    ? Object.values(FILTER_KEYS).find((d) => d.field === filterField)
    : null;

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-4 py-2">
        <div className="flex items-center gap-3">
          <span className="font-semibold">UNCWORKS</span>
          <span className="text-muted-foreground text-xs">({filtered.length})</span>

          {/* Status filter */}
          <div className="flex items-center gap-0.5 border rounded px-0.5 py-0.5">
            {(["all", "running", "failed", "succeeded"] as const).map((s) => (
              <button
                key={s}
                onClick={() => setStatusFilter(s)}
                className={`px-2 py-0.5 text-[11px] rounded transition-colors ${
                  statusFilter === s
                    ? "bg-foreground text-background font-medium"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {s}
              </button>
            ))}
          </div>
        </div>

        {/* Right side actions */}
        <div className="flex items-center gap-2">
          {activeProject && (
            <Badge variant="secondary" className="text-[10px] cursor-pointer" onClick={() => setActiveProject("")}>
              {activeProject} &times;
            </Badge>
          )}
          <button
            onClick={() => setShowArchived(!showArchived)}
            className={`text-[11px] px-2 py-0.5 rounded transition-colors ${showArchived ? "bg-muted text-foreground" : "text-muted-foreground hover:text-foreground"}`}
          >
            {showArchived ? "hiding archived" : "archived"}
          </button>
          <button
            onClick={() => { setSelectMode(!selectMode); setSelectedIds(new Set()); }}
            className={`text-[11px] px-2 py-0.5 rounded transition-colors ${selectMode ? "bg-muted text-foreground" : "text-muted-foreground hover:text-foreground"}`}
          >
            {selectMode ? "done" : "select"}
          </button>
          <Button size="sm" variant="outline" className="h-6 text-[11px] px-2" onClick={() => navigate("/new")}>
            + new
          </Button>
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

      {/* Filter bar */}
      {filterField !== null && (
        <div className="border-b px-4 py-1.5 flex items-center gap-2 bg-muted/30">
          <span className="text-xs font-mono text-muted-foreground">{activeFilterDef?.label}</span>
          <input
            autoFocus
            className="flex-1 bg-transparent text-sm outline-none"
            placeholder={activeFilterDef?.placeholder}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Escape") { setFilter(""); setFilterField(null); } }}
          />
        </div>
      )}

      {/* Mass select action bar */}
      {selectMode && selectedIds.size > 0 && (
        <div className="flex items-center gap-3 border-b bg-blue-500/10 px-4 py-1.5">
          <span className="text-xs font-medium">{selectedIds.size} selected</span>
          <Button size="sm" variant="destructive" className="h-6 text-[11px]" onClick={archiveSelected}>
            Archive
          </Button>
          <Button size="sm" variant="ghost" className="h-6 text-[11px]" onClick={() => { setSelectedIds(new Set()); setSelectMode(false); }}>
            Cancel
          </Button>
        </div>
      )}

      {/* Run rows */}
      <div className="flex-1 overflow-y-auto">
        {loading && filtered.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground">Loading...</div>
        )}
        {!loading && filtered.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground">
            {filter ? "No runs match filter" : "No runs yet — press n to create one"}
          </div>
        )}

        {viewMode === "features"
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
      <div className="border-t px-4 py-1 text-[10px] text-muted-foreground flex items-center justify-between">
        <span>j/k nav · enter open · n new · d delete · c clone · x select · p project · 1/2 view</span>
        <span>/ name · ? state · ' stage · " model</span>
      </div>
    </div>
  );
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
