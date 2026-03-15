import { useState, useEffect, useCallback } from "react";
import type { AgentRun, Repository } from "./types/agent-run";
import { useClient, mapRun } from "./hooks/useClient";
import { useWorkspaces } from "./hooks/useWorkspaces";
import type { Workspace } from "./hooks/useWorkspaces";
import { useRepoRegistry } from "./hooks/useRepoRegistry";
import { useKeyboard } from "./hooks/useKeyboard";
import { useTheme } from "./hooks/useTheme";
import { useToast } from "./components/Toast";
import { RunList } from "./components/RunList";
import { IconRail } from "./components/IconRail";
import SplitPane from "./components/SplitPane";
import CommandPalette from "./components/CommandPalette";
import RunDetail from "./components/RunDetail";
import AgentRunForm from "./components/AgentRunForm";
import ConfirmDialog from "./components/ConfirmDialog";
import WorkspaceEditor from "./components/WorkspaceEditor";
import SpecRunPage from "./pages/SpecRunPage";

/**
 * useRoute -- simple path-based routing without a router library.
 */
function useRoute(): { specRunId: string | null } {
  const [path, setPath] = useState(window.location.pathname);

  useEffect(() => {
    const onPop = () => setPath(window.location.pathname);
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);

  const match = path.match(/^\/specs\/([^/]+)\/?$/);
  return { specRunId: match ? match[1] : null };
}

type StatusFilter = "all" | "active" | "succeeded" | "failed";

const ACTIVE_PHASES = ["running", "waiting_for_input", "pending"];

export default function App() {
  const { specRunId } = useRoute();

  if (specRunId) {
    return <SpecRunPage specRunId={specRunId} />;
  }

  return <Dashboard />;
}

function Dashboard() {
  const client = useClient();
  const { toast } = useToast();
  const { toggleTheme } = useTheme();

  // Data state
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [loading, setLoading] = useState(true);

  // UI state
  const [showForm, setShowForm] = useState(false);
  const [cloneSource, setCloneSource] = useState<AgentRun | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [runToDelete, setRunToDelete] = useState<string | null>(null);
  const [editingWorkspace, setEditingWorkspace] = useState<Workspace | null | "new">(null);
  const [activeTab, setActiveTab] = useState(0);

  const { workspaces, addWorkspace, updateWorkspace, deleteWorkspace } = useWorkspaces();

  // Derive unique repos from runs
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

  // Filter runs
  const filtered = runs.filter((r) => {
    if (statusFilter !== "all") {
      if (statusFilter === "active" && !ACTIVE_PHASES.includes(r.status.phase)) return false;
      if (statusFilter === "succeeded" && r.status.phase !== "succeeded") return false;
      if (statusFilter === "failed" && r.status.phase !== "failed") return false;
    }

    return true;
  });

  // Keep selected run in sync
  const selectedRun = selectedId ? runs.find((r) => r.id === selectedId) ?? null : null;

  function handleSelectRun(run: AgentRun) {
    setSelectedId(run.id);
    setDetailOpen(true);
  }

  function handleDoubleClickRun(run: AgentRun) {
    setSelectedId(run.id);
    setDetailOpen(true);
  }

  // Keyboard system
  useKeyboard({
    runs: filtered,
    selectedId,
    setSelectedId,
    detailOpen,
    setDetailOpen,
    paletteOpen: commandPaletteOpen,
    setPaletteOpen: setCommandPaletteOpen,
    openNewRun: () => setShowForm(true),
    filter: statusFilter,
    setFilter: (f: string) => setStatusFilter(f as StatusFilter),
    activeTab,
    setActiveTab,
    tabCount: 5,
  });

  // Command palette handler
  function handleCommand(cmd: string) {
    switch (cmd) {
      case "new-run":
        setShowForm(true);
        break;
      case "toggle-theme":
        toggleTheme();
        break;
      case "filter-all":
        setStatusFilter("all");
        break;
      case "filter-active":
        setStatusFilter("active");
        break;
      case "filter-succeeded":
        setStatusFilter("succeeded");
        break;
      case "filter-failed":
        setStatusFilter("failed");
        break;
    }
  }

  function handleCommandPaletteSelectRun(run: AgentRun) {
    setSelectedId(run.id);
    setDetailOpen(true);
  }

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
    <div className="flex h-screen" style={{ backgroundColor: "var(--color-bg)", color: "var(--color-fg)" }}>
      {/* Icon Rail - 48px fixed left */}
      <IconRail
        activeFilter={statusFilter}
        onFilterChange={setStatusFilter}
        onNewRun={() => setShowForm(true)}
      />

      {/* Main content area */}
      <div className="flex flex-1 flex-col overflow-hidden min-w-0">
        {/* Header bar - 48px */}
        <header
          className="flex items-center justify-between px-4 shrink-0"
          style={{
            height: "48px",
            borderBottom: "1px solid var(--color-border)",
          }}
        >
          <span className="text-sm font-semibold" style={{ color: "var(--color-fg)" }}>AOT</span>
          <span className="text-xs" style={{ color: "var(--color-muted)" }}>
            Cmd+K search · j/k navigate · n new run
          </span>
        </header>

        {/* Split pane: RunList + Detail */}
        <SplitPane detailOpen={detailOpen && selectedRun !== null}>
          {/* Left: RunList */}
          <RunList
            runs={filtered}
            selectedId={selectedId}
            onSelect={handleSelectRun}
            onDoubleClick={handleDoubleClickRun}
            loading={loading}
          />

          {/* Right: Detail pane */}
          {selectedRun ? (
            <RunDetail
              run={selectedRun}
              onClose={() => setDetailOpen(false)}
              onCancel={handleCancel}
              onClone={handleClone}
              onSendInput={handleSendInput}
              onRefresh={fetchRuns}
            />
          ) : (
            <div />
          )}
        </SplitPane>
      </div>

      {/* Command Palette overlay */}
      <CommandPalette
        open={commandPaletteOpen}
        onClose={() => setCommandPaletteOpen(false)}
        runs={runs}
        onSelectRun={handleCommandPaletteSelectRun}
        onCommand={handleCommand}
      />

      {/* Modals */}
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
