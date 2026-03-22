import { useState, useEffect, useCallback, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import type { AgentRun, AgentRunPhase } from "../types/agent-run";
import { useClient, mapRun } from "../hooks/useClient";
import { apiFetch } from "../hooks/apiFetch";
import RunStatusBadge from "../components/RunStatusBadge";
import { Badge } from "../components/ui/badge";

type ViewMode = "features" | "all";
type FilterField = "name" | "state" | "stage" | "model" | null;

const FILTER_KEYS: Record<string, { field: FilterField; label: string; placeholder: string }> = {
  "/": { field: "name", label: "/", placeholder: "filter by name..." },
  "?": { field: "state", label: "?", placeholder: "filter by state (running, failed, succeeded)..." },
  "'": { field: "stage", label: "'", placeholder: "filter by stage (plan, execute, verify)..." },
  '"': { field: "model", label: '"', placeholder: "filter by model..." },
};

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
        onClick={(e) => {
          e.stopPropagation();
          onNavigate();
        }}
      >
        {group.feature}
      </span>
      <RunStatusBadge phase={group.phase} />
      <span className="text-xs text-muted-foreground">{group.runs.length} run{group.runs.length !== 1 ? "s" : ""}</span>
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

  // Poll every 5s
  useEffect(() => {
    fetchRuns();
    const interval = setInterval(fetchRuns, 5000);
    return () => clearInterval(interval);
  }, [fetchRuns]);

  // Derive unique project list from runs
  const projects = useMemo(() => {
    const set = new Set<string>();
    for (const r of runs) {
      if (r.spec.project) set.add(r.spec.project);
    }
    return Array.from(set).sort();
  }, [runs]);

  // Project-filtered runs
  const projectFiltered = useMemo(() => {
    if (!activeProject) return runs;
    return runs.filter((r) => r.spec.project === activeProject);
  }, [runs, activeProject]);

  // Filtered runs (archived + status + field-specific text filter)
  const filtered = useMemo(() => {
    return projectFiltered.filter((r) => {
      // Archive filter
      if (!showArchived && r.status.archived) return false;
      // Status button filter
      if (statusFilter === "running" && r.status.phase !== "running" && r.status.phase !== "waiting_for_input" && r.status.phase !== "pending") return false;
      if (statusFilter === "failed" && r.status.phase !== "failed") return false;
      if (statusFilter === "succeeded" && r.status.phase !== "succeeded") return false;
      // Text filter
      if (!filter) return true;
      const q = filter.toLowerCase();
      switch (filterField) {
        case "name":
          return (r.spec.displayName || r.name).toLowerCase().includes(q);
        case "state":
          return r.status.phase.toLowerCase().includes(q);
        case "stage":
          return (r.status.stage || "").toLowerCase().includes(q);
        case "model":
          return (r.spec.modelTier || "").toLowerCase().includes(q);
        default:
          // General search across all fields
          return (
            (r.spec.displayName || r.name).toLowerCase().includes(q) ||
            r.status.phase.toLowerCase().includes(q) ||
            (r.spec.modelTier || "").toLowerCase().includes(q) ||
            (r.spec.feature || "").toLowerCase().includes(q)
          );
      }
    });
  }, [projectFiltered, filter, filterField, statusFilter]);

  // Feature groups for features view
  const featureGroups = useMemo((): FeatureGroup[] => {
    const map = new Map<string, AgentRun[]>();
    for (const r of filtered) {
      const key = r.spec.feature || "";
      const arr = map.get(key);
      if (arr) arr.push(r);
      else map.set(key, [r]);
    }
    const groups: FeatureGroup[] = [];
    const named = Array.from(map.entries())
      .filter(([k]) => k !== "")
      .sort(([a], [b]) => a.localeCompare(b));
    for (const [feature, groupRuns] of named) {
      const prUrl = groupRuns.find((r) => r.status.prUrl)?.status.prUrl;
      groups.push({ feature, runs: groupRuns, phase: aggregatePhase(groupRuns), prUrl });
    }
    const unassigned = map.get("");
    if (unassigned) {
      const prUrl = unassigned.find((r) => r.status.prUrl)?.status.prUrl;
      groups.push({ feature: "Unassigned", runs: unassigned, phase: aggregatePhase(unassigned), prUrl });
    }
    return groups;
  }, [filtered]);

  // Flat list of visible runs in features mode (respecting collapsed state)
  const visibleRuns = useMemo((): AgentRun[] => {
    if (viewMode === "all") return filtered;
    const result: AgentRun[] = [];
    for (const group of featureGroups) {
      if (!collapsedFeatures.has(group.feature)) {
        result.push(...group.runs);
      }
    }
    return result;
  }, [viewMode, filtered, featureGroups, collapsedFeatures]);

  const toggleFeature = useCallback((feature: string) => {
    setCollapsedFeatures((prev) => {
      const next = new Set(prev);
      if (next.has(feature)) next.delete(feature);
      else next.add(feature);
      return next;
    });
  }, []);

  // Keyboard navigation
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      // Project picker takes precedence
      if (projectPickerOpen) {
        if (e.key === "Escape") {
          setProjectPickerOpen(false);
        }
        return;
      }

      if (filterField !== null) {
        if (e.key === "Escape") {
          setFilter("");
          setFilterField(null);
        }
        return;
      }

      // Check for filter shortcuts
      const filterDef = FILTER_KEYS[e.key];
      if (filterDef) {
        e.preventDefault();
        setFilterField(filterDef.field);
        return;
      }

      switch (e.key) {
        case "j":
          setSelected((s) => Math.min(s + 1, visibleRuns.length - 1));
          break;
        case "k":
          setSelected((s) => Math.max(s - 1, 0));
          break;
        case "Enter":
          if (visibleRuns[selected]) navigate(`/run/${visibleRuns[selected].id}`);
          break;
        case "n":
          navigate("/new");
          break;
        case "p":
          setProjectPickerOpen(true);
          break;
        case "1":
          setViewMode("features");
          setSelected(0);
          break;
        case "2":
          setViewMode("all");
          setSelected(0);
          break;
        case "d":
          if (visibleRuns[selected]) {
            const run = visibleRuns[selected];
            const name = run.spec.displayName || run.name;
            if (window.confirm(`Delete run ${name}?`)) {
              apiFetch(`/api/v1/runs/${run.id}`, { method: "DELETE" }).then(() => fetchRuns());
            }
          }
          break;
        case "c":
          if (visibleRuns[selected]) {
            navigate(`/new?clone=${visibleRuns[selected].id}`);
          }
          break;
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [visibleRuns, selected, filterField, projectPickerOpen, navigate, fetchRuns]);

  // Keep selection in bounds
  useEffect(() => {
    if (selected >= visibleRuns.length) setSelected(Math.max(0, visibleRuns.length - 1));
  }, [visibleRuns.length, selected]);

  function toggleSelect(id: string) {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
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

  // Render a single run row
  function RunRow({ run, index }: { run: AgentRun; index: number }) {
    const isArchived = run.status.archived;
    return (
      <div
        key={run.id}
        data-testid={`run-row-${run.id}`}
        className={`grid grid-cols-[1fr_80px_90px_55px_55px_24px_45px] gap-2 px-4 py-2 text-sm cursor-pointer transition-colors ${
          index === selected ? "bg-accent text-accent-foreground" : "hover:bg-muted/50"
        } ${isArchived ? "opacity-50" : ""}`}
        onClick={() => selectMode ? toggleSelect(run.id) : navigate(`/run/${run.id}`)}
      >
        {selectMode && (
          <input
            type="checkbox"
            checked={selectedIds.has(run.id)}
            onChange={() => toggleSelect(run.id)}
            className="accent-primary"
          />
        )}
        <span className="truncate">{run.spec.displayName || run.name}</span>
        <RunStatusBadge phase={run.status.phase} />
        <span className="text-muted-foreground text-xs truncate">{run.spec.modelTier || ""}</span>
        <span className="text-muted-foreground text-xs">
          {run.status.totalCost || "—"}
        </span>
        <span className="text-xs">
          {(run.status.totalAdditions || run.status.totalDeletions) ? (
            <>
              <span className="text-green-500">+{run.status.totalAdditions || 0}</span>
              <span className="text-red-500">/-{run.status.totalDeletions || 0}</span>
            </>
          ) : ""}
        </span>
        {run.status.prUrl ? (
          <a
            href={run.status.prUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="text-blue-500 text-xs hover:underline"
            onClick={(e) => e.stopPropagation()}
          >
            PR
          </a>
        ) : <span />}
        <span className="text-muted-foreground text-xs">{formatAge(run.createdAt)}</span>
      </div>
    );
  }

  // Track which index a run maps to in the visibleRuns flat list
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
        <div className="flex items-center gap-2">
          <span className="font-semibold">UNCWORKS</span>
          <span className="text-muted-foreground">({filtered.length})</span>
          <div className="flex items-center gap-1 ml-2">
            {(["all", "running", "failed", "succeeded"] as const).map((s) => (
              <Badge
                key={s}
                variant={statusFilter === s ? "default" : "outline"}
                className="cursor-pointer text-[10px] px-1.5 py-0"
                onClick={() => setStatusFilter(s)}
              >
                {s}
              </Badge>
            ))}
          </div>
          <span className="text-xs text-muted-foreground ml-2">
            [p] {activeProject || "all"}
          </span>
          <Badge
            variant={showArchived ? "default" : "outline"}
            className="cursor-pointer text-[10px] px-1.5 py-0 ml-2"
            onClick={() => setShowArchived(!showArchived)}
          >
            {showArchived ? "hide archived" : "show archived"}
          </Badge>
          <Badge
            variant={selectMode ? "default" : "outline"}
            className="cursor-pointer text-[10px] px-1.5 py-0"
            onClick={() => { setSelectMode(!selectMode); setSelectedIds(new Set()); }}
          >
            {selectMode ? "cancel select" : "select"}
          </Badge>
        </div>
        <span className="text-xs text-muted-foreground">/ name · ? state · ' stage · " model · n new</span>
      </div>

      {/* Project picker overlay */}
      {projectPickerOpen && (
        <div className="border-b bg-background px-4 py-2">
          <div className="text-xs text-muted-foreground mb-1">Select project:</div>
          <div
            className={`cursor-pointer px-2 py-1 text-sm rounded ${
              !activeProject ? "bg-accent text-accent-foreground" : "hover:bg-muted/50"
            }`}
            onClick={() => {
              setActiveProject("");
              setProjectPickerOpen(false);
              setSelected(0);
            }}
          >
            (all projects)
          </div>
          {projects.map((p) => (
            <div
              key={p}
              className={`cursor-pointer px-2 py-1 text-sm rounded ${
                activeProject === p ? "bg-accent text-accent-foreground" : "hover:bg-muted/50"
              }`}
              onClick={() => {
                setActiveProject(p);
                setProjectPickerOpen(false);
                setSelected(0);
              }}
            >
              {p}
            </div>
          ))}
          <div className="text-xs text-muted-foreground mt-1">esc to close</div>
        </div>
      )}

      {/* Filter bar */}
      {filterField !== null && (
        <div className="border-b px-4 py-1 flex items-center gap-2">
          <span className="text-xs text-muted-foreground font-mono">{activeFilterDef?.label}</span>
          <input
            autoFocus
            className="w-full bg-transparent text-sm outline-none"
            placeholder={activeFilterDef?.placeholder}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Escape") {
                setFilter("");
                setFilterField(null);
              }
            }}
          />
        </div>
      )}

      {/* Mass select action bar */}
      {selectMode && selectedIds.size > 0 && (
        <div className="flex items-center gap-2 border-b bg-muted/50 px-4 py-1.5">
          <span className="text-xs font-medium">{selectedIds.size} selected</span>
          <Badge variant="destructive" className="cursor-pointer text-[10px]" onClick={archiveSelected}>
            Archive
          </Badge>
          <Badge variant="outline" className="cursor-pointer text-[10px]" onClick={() => { setSelectedIds(new Set()); setSelectMode(false); }}>
            Cancel
          </Badge>
        </div>
      )}

      {/* Table header */}
      <div className={`grid grid-cols-[1fr_80px_90px_55px_55px_24px_45px] gap-2 border-b px-4 py-1 text-xs text-muted-foreground uppercase tracking-wider`}>
        {selectMode && <span />}
        <span>Name</span>
        <span>Status</span>
        <span>Model</span>
        <span>Cost</span>
        <span>+/-</span>
        <span>PR</span>
        <span>Age</span>
      </div>

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

      {/* Footer shortcuts */}
      <div className="border-t px-4 py-1 text-xs text-muted-foreground">
        j/k navigate · enter detail · n new · d delete · c clone · / name · ? state · ' stage · " model · p project
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
