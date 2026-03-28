import { useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { apiFetch } from "../hooks/apiFetch";
import { usePoll } from "../hooks/usePoll";
import { formatAge } from "../lib/format";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { Spinner } from "../components/ui/spinner";
import {
  Empty,
  EmptyHeader,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
} from "../components/ui/empty";

interface ProjectSummary {
  name: string;
  displayName: string;
  description: string;
  repos: { url: string; branch: string }[];
  configRepoReady: boolean;
  runCount: number;
  lastRunId: string;
  totalCost: string;
  createdAt: string;
}

export default function ProjectListView() {
  const navigate = useNavigate();
  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newDisplayName, setNewDisplayName] = useState("");
  const [newRepo, setNewRepo] = useState("");
  const [creating, setCreating] = useState(false);

  const fetchProjects = useCallback(async () => {
    try {
      const resp = await apiFetch("/api/v1/projects");
      if (resp.ok) {
        const data = await resp.json();
        setProjects(data);
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to load projects");
    } finally {
      setLoading(false);
    }
  }, []);

  usePoll(fetchProjects, 10000);

  async function handleCreate() {
    if (!newName.trim()) return;
    setCreating(true);
    try {
      const resp = await apiFetch("/api/v1/projects", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: newName.trim().toLowerCase().replace(/[^a-z0-9-]/g, "-"),
          displayName: newDisplayName.trim() || newName.trim(),
          repos: newRepo.trim() ? [{ url: newRepo.trim(), branch: "main" }] : [],
        }),
      });
      if (resp.ok) {
        setShowCreate(false);
        setNewName("");
        setNewDisplayName("");
        setNewRepo("");
        fetchProjects();
      }
    } catch {
      // silent
    } finally {
      setCreating(false);
    }
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <div className="flex items-center gap-3 flex-1">
          <span className="font-semibold">Projects</span>
          <span className="text-muted-foreground text-xs">({projects.length})</span>
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="ghost" onClick={() => navigate("/")}>
            Runs
          </Button>
          <Button size="sm" onClick={() => setShowCreate(!showCreate)}>
            + new project
          </Button>
        </div>
      </div>

      {/* Create form */}
      {showCreate && (
        <div className="border-b px-4 py-3 bg-muted/20 space-y-2">
          <div className="flex gap-2">
            <Input
              className="flex-1 h-8 text-sm"
              placeholder="project-name (kebab-case)"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              autoFocus
            />
            <Input
              className="flex-1 h-8 text-sm"
              placeholder="Display Name"
              value={newDisplayName}
              onChange={(e) => setNewDisplayName(e.target.value)}
            />
          </div>
          <div className="flex gap-2">
            <Input
              className="flex-1 h-8 text-sm"
              placeholder="https://github.com/org/repo (optional)"
              value={newRepo}
              onChange={(e) => setNewRepo(e.target.value)}
            />
            <Button size="sm" className="h-8" onClick={handleCreate} disabled={creating || !newName.trim()}>
              {creating ? "Creating..." : "Create"}
            </Button>
          </div>
        </div>
      )}

      {/* Project list */}
      <div className="flex-1 overflow-y-auto">
        {loading && projects.length === 0 && (
          <div className="flex h-full items-center justify-center">
            <Spinner className="text-muted-foreground" />
          </div>
        )}
        {!loading && projects.length === 0 && !showCreate && (
          <Empty className="h-full border-0">
            <EmptyHeader>
              <EmptyTitle>No projects yet</EmptyTitle>
              <EmptyDescription>Create a project to group runs, attach repos, and set defaults.</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button size="sm" onClick={() => setShowCreate(true)}>+ new project</Button>
            </EmptyContent>
          </Empty>
        )}

        {projects.map((p) => (
          <div
            key={p.name}
            className="flex items-center gap-3 px-4 py-2.5 border-b border-border/40 cursor-pointer hover:bg-muted/30 transition-colors"
            onClick={() => navigate(`/projects/${p.name}`)}
          >
            {/* Name and description */}
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-medium">{p.displayName || p.name}</span>
                {p.configRepoReady ? (
                  <Badge variant="outline" className="text-xs border-green-500/40 text-green-500">ready</Badge>
                ) : (
                  <Badge variant="secondary" className="text-xs">provisioning</Badge>
                )}
              </div>
              {p.description && (
                <span className="text-xs text-muted-foreground">{p.description}</span>
              )}
            </div>

            {/* Metadata */}
            <div className="flex items-center gap-3 shrink-0">
              {p.repos?.length > 0 && (
                <span className="text-xs text-muted-foreground">
                  {p.repos.length} repo{p.repos.length !== 1 ? "s" : ""}
                </span>
              )}
              <span className="text-xs text-muted-foreground">
                {p.runCount} run{p.runCount !== 1 ? "s" : ""}
              </span>
              {p.totalCost && (
                <span className="text-xs text-muted-foreground">{p.totalCost}</span>
              )}
              <span className="text-xs text-muted-foreground w-8 text-right">{formatAge(p.createdAt)}</span>
            </div>
          </div>
        ))}
      </div>

      {/* Footer */}
      <div className="border-t px-4 py-1 text-xs text-muted-foreground">
        click project to view specs and runs
      </div>
    </div>
  );
}

