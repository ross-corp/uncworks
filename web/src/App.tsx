import { useState, useEffect, useCallback, useRef } from "react";
import type { AgentRun, Repository } from "./types/agent-run";
import { useClient, mapRun } from "./hooks/useClient";
import { useWorkspaces } from "./hooks/useWorkspaces";
import type { Workspace } from "./hooks/useWorkspaces";
import { useRepoRegistry } from "./hooks/useRepoRegistry";
import { useKeyboardNavigation } from "./hooks/useKeyboardNavigation";
import { useToast } from "./components/Toast";
import { ThemeToggle } from "./components/ThemeToggle";
import FilterSidebar from "./components/FilterSidebar";
import type { FilterState } from "./components/FilterSidebar";
import { RunFeed } from "./components/RunFeed";
import RunDetail from "./components/RunDetail";
import AgentRunForm from "./components/AgentRunForm";
import ConfirmDialog from "./components/ConfirmDialog";
import WorkspaceEditor from "./components/WorkspaceEditor";
import { Input } from "./components/ui/input";

export default function App() {
  const client = useClient();
  const { toast } = useToast();
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [cloneSource, setCloneSource] = useState<AgentRun | null>(null);
  const [selectedRun, setSelectedRun] = useState<AgentRun | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [runToDelete, setRunToDelete] = useState<string | null>(null);
  const [editingWorkspace, setEditingWorkspace] = useState<Workspace | null | "new">(null);
  const [filters, setFilters] = useState<FilterState>({
    statuses: ["all"],
    repos: [],
    models: [],
    workspaces: [],
  });
  const searchInputRef = useRef<HTMLInputElement>(null);

  const { workspaces, addWorkspace, updateWorkspace, deleteWorkspace } = useWorkspaces();

  // Derive unique repos from runs (across all repos in each run)
  const runDerivedRepos = [...new Set(runs.flatMap((r) => r.spec.repos.map((repo) => repo.url)).filter(Boolean))];
  const { repos } = useRepoRegistry(runDerivedRepos);

  // Fetch runs from API
  const fetchRuns = useCallback(async () => {
    try {
      const result = await client.listAgentRuns();
      setRuns(result.map(mapRun));
    } catch (err) {
      console.error("Failed to fetch runs:", err);
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

  // Keep selectedRun in sync with fetched data
  useEffect(() => {
    if (selectedRun) {
      const updated = runs.find((r) => r.id === selectedRun.id);
      if (updated) {
        setSelectedRun(updated);
      }
    }
  }, [runs]); // eslint-disable-line react-hooks/exhaustive-deps


  async function handleCancel(id: string) {
    try {
      await client.cancelAgentRun(id);
      toast("Agent run cancelled", "success");
      fetchRuns();
    } catch (err) {
      console.error("Cancel failed:", err);
      toast("Failed to cancel agent run", "error");
    }
  }

  function handleDelete(id: string) {
    // Local-only for now (no delete API)
    setRuns((prev) => prev.filter((r) => r.id !== id));
    setRunToDelete(null);
  }

  async function handleSendInput(id: string, input: string) {
    try {
      await client.sendHumanInput(id, input);
      toast("Input sent to agent", "success");
      fetchRuns();
    } catch (err) {
      console.error("Send input failed:", err);
      toast("Failed to send input", "error");
    }
  }

  async function handleCreate(data: {
    name: string;
    repos: Repository[];
    workspaceName?: string;
    prompt: string;
    backend: AgentRun["spec"]["backend"];
    modelTier: string;
    ttlSeconds: number;
    specContent?: string;
    specSource?: string;
  }) {
    try {
      await client.createAgentRun({
        backend: ({ pod: "Pod", kubevirt: "KubeVirt", external: "External" } as const)[data.backend],
        repos: data.repos.map((r) => ({ url: r.url, branch: r.branch })),
        prompt: data.prompt,
        ttlSeconds: data.ttlSeconds,
        modelTier: data.modelTier,
        specContent: data.specContent,
        specSource: data.specSource,
        workspaceName: data.workspaceName,
      });
      setShowForm(false);
      toast("Agent run created", "success");
      fetchRuns();
    } catch (err) {
      console.error("Create failed:", err);
      toast("Failed to create agent run", "error");
    }
  }

  // Active statuses for the "active" filter chip
  const ACTIVE_PHASES = ["running", "waiting_for_input", "pending"];

  // Filtering: sidebar filters -> search query (additive)
  const filtered = runs.filter((r) => {
    // Status filter
    if (!filters.statuses.includes("all")) {
      const matchesStatus = filters.statuses.some((s) => {
        if (s === "active") return ACTIVE_PHASES.includes(r.status.phase);
        return r.status.phase === s;
      });
      if (!matchesStatus) return false;
    }

    // Repo filter
    if (filters.repos.length > 0) {
      const hasMatchingRepo = r.spec.repos.some((repo) => filters.repos.includes(repo.url));
      if (!hasMatchingRepo) return false;
    }

    // Model filter
    if (filters.models.length > 0) {
      if (!filters.models.includes(r.spec.modelTier)) return false;
    }

    // Workspace filter
    if (filters.workspaces.length > 0) {
      if (!r.spec.workspaceName || !filters.workspaces.includes(r.spec.workspaceName)) return false;
    }

    // Search filter (10.2, 10.3: additive with sidebar filters)
    if (searchQuery.trim()) {
      const q = searchQuery.trim().toLowerCase();
      const repoUrls = r.spec.repos.map((repo) => repo.url).join(" ");
      const haystack = [r.name, r.id, r.spec.prompt, r.status.message, repoUrls, r.spec.workspaceName ?? ""]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      if (!haystack.includes(q)) return false;
    }
    return true;
  });

  // Keyboard navigation (j/k, Enter, Escape, /)
  useKeyboardNavigation({
    runs: filtered,
    selectedRunId: selectedRun?.id ?? null,
    setSelectedRunId: setSelectedRun,
    isDetailOpen: selectedRun !== null,
    closeDetail: () => setSelectedRun(null),
    searchInputRef,
  });

  function handleClone(run: AgentRun) {
    setCloneSource(run);
    setShowForm(true);
  }

  function handleSaveWorkspace(data: { name: string; description: string; repos: Repository[] }) {
    if (editingWorkspace && editingWorkspace !== "new") {
      updateWorkspace(editingWorkspace.id, data);
    } else {
      addWorkspace(data);
    }
    setEditingWorkspace(null);
  }

  function handleDeleteWorkspace(id: string) {
    deleteWorkspace(id);
    setEditingWorkspace(null);
  }

  return (
    <div className="flex h-screen flex-col fx-flicker">
      {/* Header bar — full width (7.2) */}
      <header className="flex items-center gap-4 border-b border-border px-4 py-2 bg-background shrink-0">
        <h1 className="text-sm font-semibold tracking-widest whitespace-nowrap">
          <span className="text-primary fx-glow">AOT</span>
        </h1>
        <div className="flex-1 flex justify-center">
          <Input
            ref={searchInputRef}
            type="text"
            data-testid="search-input"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search runs..."
            className="max-w-md text-sm"
          />
        </div>
        <ThemeToggle />
      </header>

      {/* Body: sidebar + content (7.1, 7.4) */}
      <div className="flex flex-1 overflow-hidden">
        {/* FilterSidebar — fixed 280px */}
        <FilterSidebar
          runs={runs}
          filters={filters}
          onFilterChange={setFilters}
          onNewRun={() => setShowForm(true)}
        />

        {/* Content: RunFeed or RunDetail */}
        <main className="flex-1 overflow-y-auto fx-scanlines">
          {selectedRun ? (
            <RunDetail
              run={selectedRun}
              onClose={() => setSelectedRun(null)}
              onCancel={handleCancel}
              onClone={handleClone}
              onSendInput={handleSendInput}
            />
          ) : (
            <RunFeed
              runs={filtered}
              selectedRunId={null}
              onSelectRun={setSelectedRun}
              isLoading={loading}
              onNewRun={() => setShowForm(true)}
            />
          )}
        </main>
      </div>

      {showForm && (
        <AgentRunForm
          repos={repos}
          workspaces={workspaces}
          cloneSource={cloneSource ?? undefined}
          onSubmit={handleCreate}
          onCancel={() => { setShowForm(false); setCloneSource(null); }}
        />
      )}

      {editingWorkspace !== null && (
        <WorkspaceEditor
          workspace={editingWorkspace === "new" ? undefined : editingWorkspace}
          knownRepos={repos}
          onSave={handleSaveWorkspace}
          onDelete={editingWorkspace !== "new" ? handleDeleteWorkspace : undefined}
          onClose={() => setEditingWorkspace(null)}
        />
      )}

      {runToDelete && (
        <ConfirmDialog
          title="Delete Agent Run"
          message="This will permanently remove this agent run from the dashboard. This action cannot be undone."
          onConfirm={() => handleDelete(runToDelete)}
          onCancel={() => setRunToDelete(null)}
        />
      )}
    </div>
  );
}
