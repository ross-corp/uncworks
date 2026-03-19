import { useState, useEffect, useCallback } from "react";
import { Outlet } from "react-router-dom";
import type { AgentRun } from "../types/agent-run";
import { useClient, mapRun } from "../hooks/useClient";
import { useThemeNew } from "../hooks/useThemeNew";
import CommandPaletteNew from "../components/CommandPaletteNew";

/**
 * Root layout — renders the current route with command palette overlay.
 * No sidebar, no header chrome. Full screen for each view.
 */
export default function Layout() {
  // Initialize theme on mount
  useThemeNew();

  const client = useClient();
  const [runs, setRuns] = useState<AgentRun[]>([]);

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
    <div className="h-screen w-screen overflow-hidden bg-background text-foreground font-mono text-sm">
      <Outlet />
      <CommandPaletteNew runs={runs} />
    </div>
  );
}
