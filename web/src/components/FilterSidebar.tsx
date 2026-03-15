import type { AgentRun } from "../types/agent-run";
import FilterChipGroup from "./FilterChipGroup";
import { Button } from "./ui/button";

export interface FilterState {
  statuses: string[];
  repos: string[];
  models: string[];
  workspaces: string[];
}

interface FilterSidebarProps {
  runs: AgentRun[];
  filters: FilterState;
  onFilterChange: (filters: FilterState) => void;
  onNewRun: () => void;
}

const STATUS_OPTIONS = [
  { value: "all", label: "All" },
  { value: "active", label: "Active" },
  { value: "succeeded", label: "Succeeded" },
  { value: "failed", label: "Failed" },
];

function deriveRepoOptions(runs: AgentRun[]): { value: string; label: string }[] {
  const urls = [...new Set(runs.flatMap((r) => r.spec.repos.map((repo) => repo.url)).filter(Boolean))];
  return urls.map((url) => ({
    value: url,
    label: url.replace(/\.git$/, "").split("/").pop() ?? url,
  }));
}

function deriveModelOptions(runs: AgentRun[]): { value: string; label: string }[] {
  const tiers = [...new Set(runs.map((r) => r.spec.modelTier).filter(Boolean))];
  return tiers.map((t) => ({ value: t, label: t }));
}

function deriveWorkspaceOptions(runs: AgentRun[]): { value: string; label: string }[] {
  const names = [...new Set(runs.map((r) => r.spec.workspaceName).filter((n): n is string => !!n))];
  return names.map((n) => ({ value: n, label: n }));
}

export default function FilterSidebar({
  runs,
  filters,
  onFilterChange,
  onNewRun,
}: FilterSidebarProps) {
  const repoOptions = deriveRepoOptions(runs);
  const modelOptions = deriveModelOptions(runs);
  const workspaceOptions = deriveWorkspaceOptions(runs);

  function handleStatusToggle(value: string) {
    if (value === "all") {
      onFilterChange({ ...filters, statuses: ["all"] });
      return;
    }
    let next = filters.statuses.filter((s) => s !== "all");
    if (next.includes(value)) {
      next = next.filter((s) => s !== value);
    } else {
      next = [...next, value];
    }
    if (next.length === 0) next = ["all"];
    onFilterChange({ ...filters, statuses: next });
  }

  function handleRepoToggle(value: string) {
    let next: string[];
    if (filters.repos.includes(value)) {
      next = filters.repos.filter((r) => r !== value);
    } else {
      next = [...filters.repos, value];
    }
    onFilterChange({ ...filters, repos: next });
  }

  function handleRepoRemove(value: string) {
    onFilterChange({ ...filters, repos: filters.repos.filter((r) => r !== value) });
  }

  function handleModelToggle(value: string) {
    let next: string[];
    if (filters.models.includes(value)) {
      next = filters.models.filter((m) => m !== value);
    } else {
      next = [...filters.models, value];
    }
    onFilterChange({ ...filters, models: next });
  }

  function handleWorkspaceToggle(value: string) {
    let next: string[];
    if (filters.workspaces.includes(value)) {
      next = filters.workspaces.filter((w) => w !== value);
    } else {
      next = [...filters.workspaces, value];
    }
    onFilterChange({ ...filters, workspaces: next });
  }

  return (
    <aside className="flex h-full w-[280px] flex-col border-r border-border bg-background">
      {/* + New Run button */}
      <div className="px-4 pt-4 pb-3">
        <Button
          data-testid="sidebar-new-run"
          onClick={onNewRun}
          className="w-full text-xs"
          size="sm"
        >
          + New Run
        </Button>
      </div>

      {/* Filter groups */}
      <div className="flex-1 overflow-y-auto px-4 pb-4 space-y-5">
        <FilterChipGroup
          label="Status"
          options={STATUS_OPTIONS}
          selected={filters.statuses}
          onToggle={handleStatusToggle}
        />

        {repoOptions.length > 0 && (
          <FilterChipGroup
            label="Repos"
            options={repoOptions}
            selected={filters.repos}
            onToggle={handleRepoToggle}
            onRemove={handleRepoRemove}
          />
        )}

        {modelOptions.length > 0 && (
          <FilterChipGroup
            label="Model"
            options={modelOptions}
            selected={filters.models}
            onToggle={handleModelToggle}
          />
        )}

        {workspaceOptions.length > 0 && (
          <FilterChipGroup
            label="Workspace"
            options={workspaceOptions}
            selected={filters.workspaces}
            onToggle={handleWorkspaceToggle}
          />
        )}
      </div>

    </aside>
  );
}
