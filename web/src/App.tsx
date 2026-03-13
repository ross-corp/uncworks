import { useState, useEffect, useCallback } from "react";
import type { AgentRun } from "./types/agent-run";
import { useClient, mapRun } from "./hooks/useClient";
import { useToast } from "./components/Toast";
import Layout from "./components/Layout";
import Sidebar from "./components/Sidebar";
import AgentRunTable from "./components/AgentRunTable";
import AgentRunDetailPanel from "./components/AgentRunDetailPanel";
import AgentRunForm from "./components/AgentRunForm";
import ReposView from "./components/ReposView";
import EventsView from "./components/EventsView";
import ConfirmDialog from "./components/ConfirmDialog";

export default function App() {
  const client = useClient();
  const { toast } = useToast();
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedRepo, setSelectedRepo] = useState<string | null>(null);
  const [phaseFilter, setPhaseFilter] = useState("all");
  const [showForm, setShowForm] = useState(false);
  const [selectedRun, setSelectedRun] = useState<AgentRun | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [activeView, setActiveView] = useState<"runs" | "repos" | "events">("runs");
  const [runToDelete, setRunToDelete] = useState<string | null>(null);

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
    repoURL: string;
    branch: string;
    prompt: string;
    backend: AgentRun["spec"]["backend"];
    modelTier: string;
    ttlSeconds: number;
  }) {
    try {
      await client.createAgentRun({
        backend: ({ pod: "Pod", kubevirt: "KubeVirt", external: "External" } as const)[data.backend],
        repos: [{ url: data.repoURL, branch: data.branch }],
        prompt: data.prompt,
        ttlSeconds: data.ttlSeconds,
        modelTier: data.modelTier,
      });
      setShowForm(false);
      toast("Agent run created", "success");
      fetchRuns();
    } catch (err) {
      console.error("Create failed:", err);
      toast("Failed to create agent run", "error");
    }
  }

  // Derive unique repos from runs
  const repos = [...new Set(runs.map((r) => r.spec.repoURL).filter(Boolean))];

  // Filtering
  const filtered = runs.filter((r) => {
    if (selectedRepo && r.spec.repoURL !== selectedRepo) return false;
    if (phaseFilter !== "all" && r.status.phase !== phaseFilter) return false;
    if (searchQuery.trim()) {
      const q = searchQuery.trim().toLowerCase();
      const haystack = [r.name, r.id, r.spec.prompt, r.status.message, r.spec.repoURL]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      if (!haystack.includes(q)) return false;
    }
    return true;
  });

  return (
    <>
      <Layout
        onNewRun={() => setShowForm(true)}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        activeView={activeView}
        sidebar={
          <Sidebar
            runs={runs}
            repos={repos}
            selectedRepo={selectedRepo}
            onSelectRepo={(url) => {
              setSelectedRepo(url);
              setActiveView("runs");
            }}
            phaseFilter={phaseFilter}
            onPhaseFilter={(f) => {
              setPhaseFilter(f);
              setActiveView("runs");
            }}
            onOpenRepos={() => {
              setActiveView("repos");
              setSelectedRun(null);
            }}
            onOpenEvents={() => {
              setActiveView("events");
              setSelectedRun(null);
            }}
          />
        }
        detailPanel={
          activeView === "runs" && selectedRun ? (
            <AgentRunDetailPanel
              run={selectedRun}
              onClose={() => setSelectedRun(null)}
              onCancel={handleCancel}
              onSendInput={handleSendInput}
            />
          ) : undefined
        }
      >
        {activeView === "repos" ? (
          <ReposView repos={repos} />
        ) : activeView === "events" ? (
          <EventsView runs={runs} />
        ) : (
          <AgentRunTable
            runs={filtered}
            selectedRunId={selectedRun?.id}
            onSelect={setSelectedRun}
            onCancel={handleCancel}
            onDelete={setRunToDelete}
            loading={loading}
            onNewRun={() => setShowForm(true)}
          />
        )}
      </Layout>

      {showForm && (
        <AgentRunForm
          repos={repos}
          onSubmit={handleCreate}
          onCancel={() => setShowForm(false)}
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
    </>
  );
}
