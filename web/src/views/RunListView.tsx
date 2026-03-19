import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import type { AgentRun } from "../types/agent-run";
import { useClient, mapRun } from "../hooks/useClient";
import RunStatusBadge from "../components/RunStatusBadge";

export default function RunListView() {
  const client = useClient();
  const navigate = useNavigate();
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [selected, setSelected] = useState(0);
  const [filter, setFilter] = useState("");
  const [filterMode, setFilterMode] = useState(false);
  const [loading, setLoading] = useState(true);

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

  // Filtered runs
  const filtered = runs.filter((r) => {
    if (!filter) return true;
    const q = filter.toLowerCase();
    return (
      (r.spec.displayName || r.name).toLowerCase().includes(q) ||
      r.status.phase.toLowerCase().includes(q) ||
      (r.spec.modelTier || "").toLowerCase().includes(q)
    );
  });

  // Keyboard navigation
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (filterMode) {
        if (e.key === "Escape") {
          setFilter("");
          setFilterMode(false);
        }
        return;
      }

      switch (e.key) {
        case "j":
          setSelected((s) => Math.min(s + 1, filtered.length - 1));
          break;
        case "k":
          setSelected((s) => Math.max(s - 1, 0));
          break;
        case "Enter":
          if (filtered[selected]) navigate(`/run/${filtered[selected].id}`);
          break;
        case "n":
          navigate("/new");
          break;
        case "/":
          e.preventDefault();
          setFilterMode(true);
          break;
        case "1":
          setFilter("");
          break;
        case "2":
          setFilter("running");
          break;
        case "3":
          setFilter("succeeded");
          break;
        case "4":
          setFilter("failed");
          break;
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [filtered, selected, filterMode, navigate]);

  // Keep selection in bounds
  useEffect(() => {
    if (selected >= filtered.length) setSelected(Math.max(0, filtered.length - 1));
  }, [filtered.length, selected]);

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-4 py-2">
        <div className="flex items-center gap-2">
          <span className="font-semibold">AOT</span>
          <span className="text-muted-foreground">Runs ({filtered.length})</span>
        </div>
        <span className="text-xs text-muted-foreground">⌘K search · n new · ? help</span>
      </div>

      {/* Filter bar */}
      {filterMode && (
        <div className="border-b px-4 py-1">
          <input
            autoFocus
            className="w-full bg-transparent text-sm outline-none"
            placeholder="/ filter runs..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Escape") {
                setFilter("");
                setFilterMode(false);
              }
            }}
          />
        </div>
      )}

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
        {loading && filtered.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground">Loading...</div>
        )}
        {!loading && filtered.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground">
            {filter ? "No runs match filter" : "No runs yet — press n to create one"}
          </div>
        )}
        {filtered.map((run, i) => (
          <div
            key={run.id}
            data-testid={`run-row-${run.id}`}
            className={`grid grid-cols-[1fr_100px_80px_100px_70px] gap-2 px-4 py-2 text-sm cursor-pointer transition-colors ${
              i === selected ? "bg-accent text-accent-foreground" : "hover:bg-muted/50"
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
        j/k navigate · enter detail · n new · / filter · 1-4 quick filter
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
