import { useState, useEffect } from "react";
import type { AgentRun } from "./types/agent-run";
import { MOCK_AGENT_RUNS, MOCK_REPOS } from "./data/mock";
import Layout from "./components/Layout";
import Sidebar from "./components/Sidebar";
import AgentRunTable from "./components/AgentRunTable";
import AgentRunDetailPanel from "./components/AgentRunDetailPanel";
import AgentRunForm from "./components/AgentRunForm";
import ReposView from "./components/ReposView";
import EventsView from "./components/EventsView";
import ConfirmDialog from "./components/ConfirmDialog";

export default function App() {
  const [runs, setRuns] = useState<AgentRun[]>(MOCK_AGENT_RUNS);
  const [selectedRepo, setSelectedRepo] = useState<string | null>(null);
  const [phaseFilter, setPhaseFilter] = useState("all");
  const [showForm, setShowForm] = useState(false);
  const [selectedRun, setSelectedRun] = useState<AgentRun | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [activeView, setActiveView] = useState<"runs" | "repos" | "events">("runs");
  const [runToDelete, setRunToDelete] = useState<string | null>(null);

  // Keep selectedRun in sync
  useEffect(() => {
    if (selectedRun) {
      const updated = runs.find((r) => r.id === selectedRun.id);
      if (updated) {
        setSelectedRun(updated);
      } else {
        setSelectedRun(null);
      }
    }
  }, [runs]); // eslint-disable-line react-hooks/exhaustive-deps

  function handleCancel(id: string) {
    setRuns((prev) =>
      prev.map((r) =>
        r.id === id
          ? {
              ...r,
              status: {
                ...r.status,
                phase: "cancelled" as const,
                message: "Cancelled by user",
                completedAt: new Date().toISOString(),
              },
            }
          : r
      )
    );
  }

  function handleDelete(id: string) {
    setRuns((prev) => prev.filter((r) => r.id !== id));
    setRunToDelete(null);
  }

  function handleSendInput(id: string, input: string) {
    setRuns((prev) =>
      prev.map((r) =>
        r.id === id
          ? {
              ...r,
              status: {
                ...r.status,
                phase: "running" as const,
                message: `Received input: "${input.slice(0, 50)}..."`,
              },
            }
          : r
      )
    );
  }

  function handleCreate(data: {
    name: string;
    repoURL: string;
    branch: string;
    prompt: string;
    backend: AgentRun["spec"]["backend"];
    modelTier: AgentRun["spec"]["modelTier"];
    ttlSeconds: number;
  }) {
    const newRun: AgentRun = {
      id: `ar-${String(runs.length + 1).padStart(3, "0")}`,
      name: data.name,
      spec: {
        backend: data.backend,
        repoURL: data.repoURL,
        branch: data.branch,
        prompt: data.prompt,
        devboxConfig: "",
        ttlSeconds: data.ttlSeconds,
        envVars: {},
        modelTier: data.modelTier,
      },
      status: {
        phase: "pending",
        message: "",
        podName: "",
        traceID: "",
        startedAt: "",
        completedAt: "",
      },
      createdAt: new Date().toISOString(),
    };
    setRuns((prev) => [newRun, ...prev]);
    setShowForm(false);
  }

  // Filtering: repo → phase → search
  const filtered = runs.filter((r) => {
    if (selectedRepo && r.spec.repoURL !== selectedRepo) return false;
    if (phaseFilter !== "all" && r.status.phase !== phaseFilter) return false;
    if (searchQuery.trim()) {
      const q = searchQuery.trim().toLowerCase();
      const haystack = [
        r.name,
        r.id,
        r.spec.prompt,
        r.status.message,
        r.spec.repoURL,
      ]
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
            repos={MOCK_REPOS}
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
          <ReposView repos={MOCK_REPOS} />
        ) : activeView === "events" ? (
          <EventsView runs={runs} />
        ) : (
          <AgentRunTable
            runs={filtered}
            selectedRunId={selectedRun?.id}
            onSelect={setSelectedRun}
            onCancel={handleCancel}
            onDelete={setRunToDelete}
          />
        )}
      </Layout>

      {showForm && (
        <AgentRunForm
          repos={MOCK_REPOS}
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
