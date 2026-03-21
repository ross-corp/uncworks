import { useState, useEffect, useCallback } from "react";
import { Outlet, useLocation } from "react-router-dom";
import type { AgentRun } from "../types/agent-run";
import { useClient, mapRun } from "../hooks/useClient";
import { useThemeNew } from "../hooks/useThemeNew";
import CommandPaletteNew from "../components/CommandPaletteNew";

/**
 * Root layout — renders the current route with command palette overlay.
 * No sidebar, no header chrome. Full screen for each view.
 */
export default function Layout() {
  const { mode, toggleMode } = useThemeNew();

  // Resolve current mode for theme toggle icon
  const resolvedMode =
    mode === "system"
      ? window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light"
      : mode;

  const client = useClient();
  const location = useLocation();
  const [runs, setRuns] = useState<AgentRun[]>([]);

  // Extract run ID from /run/:id path
  const runIdMatch = location.pathname.match(/^\/run\/([^/]+)/);
  const selectedRunId = runIdMatch?.[1];

  const fetchRuns = useCallback(async () => {
    try {
      const result = await client.listAgentRuns();
      setRuns(result.map(mapRun));
    } catch {
      // silent
    }
  }, [client]);

  useEffect(() => {
    fetchRuns();
    const interval = setInterval(fetchRuns, 10000);
    return () => clearInterval(interval);
  }, [fetchRuns]);

  return (
    <div className="h-screen w-screen overflow-hidden bg-background text-foreground font-mono text-sm flex flex-col">
      <div className="flex-1 min-h-0">
        <Outlet />
      </div>
      <div className="flex items-center justify-end border-t px-4 py-1">
        <button
          onClick={toggleMode}
          className="px-1.5 py-0.5 text-sm text-muted-foreground hover:text-foreground"
          title={`Switch to ${resolvedMode === "dark" ? "light" : "dark"} mode`}
        >
          {resolvedMode === "dark" ? "\u2600" : "\u263E"}
        </button>
      </div>
      <CommandPaletteNew runs={runs} selectedRunId={selectedRunId} />
    </div>
  );
}
