import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Command } from "cmdk";
import type { AgentRun } from "../types/agent-run";
import { useThemeNew, THEMES } from "../hooks/useThemeNew";
import { apiFetch } from "../hooks/apiFetch";

import "./command-palette.css";

interface Props {
  runs: AgentRun[];
  selectedRunId?: string;
}

export default function CommandPaletteNew({ runs, selectedRunId }: Props) {
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const { setTheme, toggleMode, theme } = useThemeNew();

  // ⌘K to toggle
  useEffect(() => {
    function handler(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setOpen((o) => !o);
      }
    }
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  function runAction(fn: () => void) {
    fn();
    setOpen(false);
  }

  return (
    <Command.Dialog
      open={open}
      onOpenChange={setOpen}
      label="Command palette"
      className="cmdk-dialog"
    >
      <Command.Input placeholder="Type a command or search..." className="cmdk-input" />
      <Command.List className="cmdk-list">
        <Command.Empty>No results found.</Command.Empty>

        {/* Navigation */}
        <Command.Group heading="Navigation">
          <Command.Item onSelect={() => runAction(() => navigate("/"))}>
            Go to Runs
          </Command.Item>
          <Command.Item onSelect={() => runAction(() => navigate("/new"))}>
            New Run
          </Command.Item>
        </Command.Group>

        {/* Actions */}
        {selectedRunId && (() => {
          const selectedRun = runs.find((r) => r.id === selectedRunId);
          return selectedRun ? (
            <Command.Group heading="Actions">
              {selectedRun.status.phase === "running" && (
                <Command.Item
                  onSelect={() =>
                    runAction(() => {
                      apiFetch(`/api/v1/runs/${selectedRunId}/cancel`, { method: "POST" });
                    })
                  }
                >
                  Cancel selected run
                </Command.Item>
              )}
              <Command.Item
                onSelect={() => runAction(() => navigate(`/new?clone=${selectedRunId}`))}
              >
                Clone run
              </Command.Item>
            </Command.Group>
          ) : null;
        })()}

        {/* Runs */}
        {runs.length > 0 && (
          <Command.Group heading="Runs">
            {runs.slice(0, 10).map((run) => (
              <Command.Item
                key={run.id}
                value={`${run.spec.displayName || run.name} ${run.id}`}
                onSelect={() => runAction(() => navigate(`/run/${run.id}`))}
              >
                {run.spec.displayName || run.name}
                <span className="ml-2 text-muted-foreground text-xs">{run.status.phase}</span>
              </Command.Item>
            ))}
          </Command.Group>
        )}

        {/* Theme */}
        <Command.Group heading="Theme">
          <Command.Item onSelect={() => runAction(toggleMode)}>
            Toggle dark mode
          </Command.Item>
          {THEMES.map((t) => (
            <Command.Item
              key={t}
              onSelect={() => runAction(() => setTheme(t))}
              value={`theme ${t}`}
            >
              {t === theme ? `● ${t}` : `  ${t}`}
            </Command.Item>
          ))}
        </Command.Group>
      </Command.List>
    </Command.Dialog>
  );
}
