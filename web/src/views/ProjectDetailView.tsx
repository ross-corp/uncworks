import { useState, useEffect, useCallback, useMemo } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { toast } from "sonner";
import { apiFetch } from "../hooks/apiFetch";
import { useClient, mapRun } from "../hooks/useClient";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../components/ui/tabs";
import MarkdownEditor from "../components/MarkdownEditor";
import RunStatusBadge from "../components/RunStatusBadge";
import { formatAge } from "../lib/format";
import type { AgentRun } from "../types/agent-run";
import { MODEL_OPTIONS } from "../types/agent-run";
import CustomSelect from "../components/CustomSelect";
import { useCopilotContext, useCopilotContextValue } from "../hooks/useCopilotContext";

interface ProjectDetail {
  name: string;
  displayName: string;
  description: string;
  repos: { url: string; branch: string }[];
  devbox?: { packages: string[] };
  defaults?: {
    modelTier?: string;
    manageModelTier?: string;
    implementModelTier?: string;
    ttlSeconds?: number;
    orchestrationMode?: string;
    autoPush?: boolean;
    autoPR?: boolean;
    prBaseBranch?: string;
  };
  configRepoReady: boolean;
  configRepoURL: string;
  configRepoMessage?: string;
  runCount: number;
  totalCost: string;
}


export default function ProjectDetailView() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const client = useClient();
  const [project, setProject] = useState<ProjectDetail | null>(null);
  const [files, setFiles] = useState<string[]>([]);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState("");
  const [editedContent, setEditedContent] = useState("");
  const [saving, setSaving] = useState(false);
  const [improving, setImproving] = useState(false);
  const [tab, setTab] = useState<"specs" | "runs" | "settings">("specs");

  // Settings editing state
  const [editDescription, setEditDescription] = useState("");
  const [settingsDirty, setSettingsDirty] = useState(false);
  const [savingSettings, setSavingSettings] = useState(false);

  // Rename state
  const [renaming, setRenaming] = useState(false);
  const [newName, setNewName] = useState("");
  const [renameSaving, setRenameSaving] = useState(false);

  const { setOpen: setCopilotOpen } = useCopilotContextValue();

  // Register page context for global CopilotPanel.
  const copilotCtx = useMemo(() => {
    if (tab === "specs" && selectedFile && editedContent) {
      return { type: "spec" as const, content: editedContent, label: selectedFile };
    }
    if (tab !== "specs" && project) {
      return {
        type: "project" as const,
        content: project.description || project.name,
        label: project.name,
      };
    }
    return null;
  }, [tab, selectedFile, editedContent, project]);
  useCopilotContext(copilotCtx);

  // Devbox packages state
  const [editDevboxPackages, setEditDevboxPackages] = useState<string[]>([]);
  const [newPackage, setNewPackage] = useState("");

  // Run defaults state
  const [editDefaults, setEditDefaults] = useState<{
    modelTier: string;
    manageModelTier: string;
    implementModelTier: string;
    ttlSeconds: string;
    orchestrationMode: string;
    autoPush: boolean;
    autoPR: boolean;
    prBaseBranch: string;
  }>({ modelTier: "", manageModelTier: "", implementModelTier: "", ttlSeconds: "", orchestrationMode: "", autoPush: false, autoPR: false, prBaseBranch: "" });

  // New spec creation
  const [showNewSpec, setShowNewSpec] = useState(false);
  const [newSpecName, setNewSpecName] = useState("");
  const [creatingSpec, setCreatingSpec] = useState(false);

  // Runs tab
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [runsLoaded, setRunsLoaded] = useState(false);

  const fetchFiles = useCallback(async () => {
    if (!name) return;
    try {
      const resp = await apiFetch(`/api/v1/projects/${name}/files`);
      if (resp.ok) setFiles(await resp.json());
    } catch (e) {
      toast.error(`Failed to load files: ${e instanceof Error ? e.message : String(e)}`);
    }
  }, [name]);

  useEffect(() => {
    let cancelled = false;

    const fetch = async () => {
      if (!name) return;
      try {
        const resp = await apiFetch(`/api/v1/projects/${name}`);
        if (resp.status === 404) {
          if (!cancelled) navigate("/projects", { replace: true });
          return;
        }
        if (resp.ok) {
          const data = await resp.json();
          if (!cancelled) {
            setProject(data);
            setEditDescription(data.description || "");
            setEditDevboxPackages(data.devbox?.packages || []);
            setEditDefaults({
              modelTier: data.defaults?.modelTier || "",
              manageModelTier: data.defaults?.manageModelTier || "",
              implementModelTier: data.defaults?.implementModelTier || "",
              ttlSeconds: data.defaults?.ttlSeconds != null ? String(data.defaults.ttlSeconds) : "",
              orchestrationMode: data.defaults?.orchestrationMode || "",
              autoPush: data.defaults?.autoPush || false,
              autoPR: data.defaults?.autoPR || false,
              prBaseBranch: data.defaults?.prBaseBranch || "",
            });
            setSettingsDirty(false);
          }
        }
      } catch (e) {
        if (!cancelled) toast.error(`Failed to load project: ${e instanceof Error ? e.message : String(e)}`);
      }

      try {
        const resp = await apiFetch(`/api/v1/projects/${name}/files`);
        if (resp.ok) {
          const data = await resp.json();
          if (!cancelled) setFiles(data);
        }
      } catch (e) {
        if (!cancelled) toast.error(`Failed to load files: ${e instanceof Error ? e.message : String(e)}`);
      }
    };

    fetch();

    return () => {
      cancelled = true;
    };
  }, [name]);

  useEffect(() => {
    if (tab !== "runs") return;

    let cancelled = false;

    const doFetch = async () => {
      try {
        // Use the ConnectRPC client — listAgentRuns() already returns mapped AgentRun[].
        const raw = await client.listAgentRuns();
        if (!cancelled) {
          setRuns(raw.map(mapRun).filter((r) => r.spec.projectRef === name || r.spec.project === name));
          setRunsLoaded(true);
        }
      } catch (e) {
        if (!cancelled) toast.error(`Failed to load runs: ${e instanceof Error ? e.message : String(e)}`);
      }
    };

    doFetch();
    const i = setInterval(() => {
      if (!cancelled) doFetch();
    }, 10000);

    return () => {
      cancelled = true;
      clearInterval(i);
    };
  }, [tab, name, client]);

  async function loadFile(path: string) {
    if (!name) return;
    // Warn before discarding unsaved edits
    if (editedContent !== fileContent && selectedFile !== path) {
      const ok = window.confirm(`You have unsaved changes in "${selectedFile}". Discard them?`);
      if (!ok) return;
    }
    setSelectedFile(path);
    try {
      const resp = await apiFetch(`/api/v1/projects/${name}/files/${path}`);
      if (resp.ok) {
        const data = await resp.json();
        setFileContent(data.content);
        setEditedContent(data.content);
      }
    } catch (e) {
      toast.error(`Failed to load file: ${e instanceof Error ? e.message : String(e)}`);
    }
  }

  async function saveFile() {
    if (!name || !selectedFile) return;
    setSaving(true);
    try {
      const resp = await apiFetch(`/api/v1/projects/${name}/files/${selectedFile}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ content: editedContent, commitMessage: `update ${selectedFile}` }),
      });
      if (resp.ok) {
        // Only clear the modified badge when the server confirmed the write
        setFileContent(editedContent);
      } else {
        const body = await resp.json().catch(() => ({}));
        toast.error(`Failed to save: ${(body as { error?: string }).error ?? resp.statusText}`);
      }
    } catch (e) {
      toast.error(`Failed to save file: ${e instanceof Error ? e.message : String(e)}`);
    }
    setSaving(false);
  }

  async function improveWithAI() {
    if (!editedContent.trim()) return;
    setImproving(true);
    try {
      const kind = selectedFile?.endsWith("spec.md") ? "spec" : "prompt";
      const resp = await apiFetch("/api/v1/improve-text", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text: editedContent, kind }),
      });
      if (resp.ok) {
        const data = await resp.json();
        if (data.improved) setEditedContent(data.improved);
      }
    } catch (e) {
      toast.error(`Improve with AI failed: ${e instanceof Error ? e.message : String(e)}`);
    }
    setImproving(false);
  }

  async function createSpec() {
    if (!name || !newSpecName.trim()) return;
    const specSlug = newSpecName.trim().toLowerCase().replace(/[^a-z0-9-]/g, "-");
    const path = `openspec/specs/${specSlug}/spec.md`;
    setCreatingSpec(true);
    try {
      const resp = await apiFetch(`/api/v1/projects/${name}/files/${path}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          content: `# ${newSpecName.trim()}\n\n## Requirements\n\n- \n\n## Acceptance Criteria\n\n- \n`,
          commitMessage: `create spec: ${specSlug}`,
        }),
      });
      if (!resp.ok) {
        const body = await resp.json().catch(() => ({}));
        throw new Error((body as { error?: string }).error ?? resp.statusText);
      }
      setShowNewSpec(false);
      setNewSpecName("");
      await fetchFiles();
      // Auto-select and open the newly created spec
      await loadFile(path);
    } catch (e) {
      toast.error(`Failed to create spec: ${e instanceof Error ? e.message : String(e)}`);
    }
    setCreatingSpec(false);
  }

  async function saveSettings() {
    if (!name) return;
    setSavingSettings(true);
    try {
      const resp = await apiFetch(`/api/v1/projects/${name}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          description: editDescription,
          devbox: editDevboxPackages.length > 0 ? { packages: editDevboxPackages } : undefined,
          defaults: {
            modelTier: editDefaults.modelTier || undefined,
            manageModelTier: editDefaults.manageModelTier || undefined,
            implementModelTier: editDefaults.implementModelTier || undefined,
            ttlSeconds: editDefaults.ttlSeconds ? parseInt(editDefaults.ttlSeconds) : undefined,
            orchestrationMode: editDefaults.orchestrationMode || undefined,
            autoPush: editDefaults.autoPush || undefined,
            autoPR: editDefaults.autoPR || undefined,
            prBaseBranch: editDefaults.prBaseBranch || undefined,
          },
        }),
      });
      if (resp.ok) {
        setProject(await resp.json());
        setSettingsDirty(false);
      }
    } catch (e) {
      toast.error(`Failed to save settings: ${e instanceof Error ? e.message : String(e)}`);
    }
    setSavingSettings(false);
  }

  async function handleRename() {
    if (!name || !newName.trim()) return;
    const slug = newName.trim().toLowerCase().replace(/[^a-z0-9-]/g, "-");
    if (slug === name) { setRenaming(false); return; }
    setRenameSaving(true);
    try {
      // Create new project with same settings
      const createResp = await apiFetch("/api/v1/projects", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: slug }),
      });
      if (!createResp.ok) throw new Error(await createResp.text());
      // Copy current settings to new project
      if (project) {
        await apiFetch(`/api/v1/projects/${slug}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            description: project.description,
            devbox: project.devbox,
            defaults: project.defaults,
          }),
        });
      }
      // Delete old project
      await apiFetch(`/api/v1/projects/${name}`, { method: "DELETE" });
      navigate(`/projects/${slug}`, { replace: true });
    } catch (e) {
      toast.error(`Rename failed: ${e instanceof Error ? e.message : String(e)}`);
    }
    setRenameSaving(false);
  }

  function addPackage() {
    const pkg = newPackage.trim();
    if (pkg && !editDevboxPackages.includes(pkg)) {
      setEditDevboxPackages([...editDevboxPackages, pkg]);
      setNewPackage("");
      setSettingsDirty(true);
    }
  }
  function removePackage(pkg: string) {
    setEditDevboxPackages(editDevboxPackages.filter((p) => p !== pkg));
    setSettingsDirty(true);
  }

  const specFiles = files.filter((f) => f.startsWith("openspec/specs/"));
  const otherFiles = files.filter((f) => !f.startsWith("openspec/specs/"));
  const hasChanges = editedContent !== fileContent;
  // Non-spec files (config, dotfiles) are read-only in this editor
  const isReadOnly = selectedFile !== null && !selectedFile.startsWith("openspec/specs/");

  if (!project) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Loading project...
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="h-12 border-b flex items-center px-4 gap-2 justify-between">
        <div className="flex flex-col gap-0.5">
          <div className="text-xs text-muted-foreground">
            <Link to="/projects" className="hover:text-foreground transition-colors">Projects</Link>
            {" / "}
            <span>{project.name}</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="font-semibold">{project.name}</span>
            {project.configRepoReady ? (
              <Badge variant="outline" className="text-xs border-green-500/40 text-green-500">ready</Badge>
            ) : project.configRepoMessage ? (
              <Badge variant="secondary" className="text-xs" title={project.configRepoMessage}>
                provisioning
              </Badge>
            ) : (
              <Badge variant="outline" className="text-xs border-green-500/40 text-green-500">active</Badge>
            )}
            <span className="text-xs text-muted-foreground">{runsLoaded ? runs.length : project.runCount} runs</span>
            {project.totalCost && (
              <span className="text-xs text-muted-foreground">{project.totalCost}</span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {tab === "specs" && (
            <Button size="sm" variant="ghost" onClick={() => setShowNewSpec(true)}>
              + new spec
            </Button>
          )}
          <Button size="sm" variant="ghost" onClick={() => navigate(`/new?project=${name}`)}>
            + new run
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <Tabs value={tab} onValueChange={(v) => setTab(v as typeof tab)} className="flex flex-col flex-1 min-h-0 gap-0">
        <TabsList className="rounded-none border-b shrink-0">
          <TabsTrigger value="specs">Specs</TabsTrigger>
          <TabsTrigger value="runs">Runs</TabsTrigger>
          <TabsTrigger value="settings">Settings</TabsTrigger>
        </TabsList>

        {/* Specs tab */}
        <TabsContent value="specs" className="flex flex-1 min-h-0 mt-0 data-[state=inactive]:hidden">
          {/* Spec file tree */}
          <div className="w-56 border-r overflow-y-auto overscroll-none p-2">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-muted-foreground uppercase tracking-wider">Specs</span>
            </div>

            {showNewSpec && (
              <div className="mb-2 space-y-1">
                <Input
                  className="h-6 text-xs"
                  placeholder="spec-name"
                  value={newSpecName}
                  onChange={(e) => setNewSpecName(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter") createSpec(); if (e.key === "Escape") setShowNewSpec(false); }}
                  autoFocus
                />
                <div className="flex gap-1">
                  <Button size="sm" className="text-xs flex-1" onClick={createSpec} disabled={creatingSpec || !newSpecName.trim()}>
                    {creatingSpec ? "..." : "Create"}
                  </Button>
                  <Button size="sm" variant="ghost" className="text-xs" onClick={() => setShowNewSpec(false)}>
                    Cancel
                  </Button>
                </div>
              </div>
            )}

            {specFiles.length === 0 && !showNewSpec && (
              <div className="text-xs text-muted-foreground p-2">
                {!project.configRepoReady && project.configRepoMessage
                  ? "Config repo is still provisioning. Specs will appear here once it is ready."
                  : "No specs yet"}
              </div>
            )}
            {specFiles.map((f) => {
              const label = f.replace("openspec/specs/", "").replace("/spec.md", "");
              return (
                <div
                  key={f}
                  className={`text-xs px-2 py-1 cursor-pointer rounded transition-colors ${
                    selectedFile === f ? "bg-accent text-accent-foreground" : "hover:bg-muted/50"
                  }`}
                  onClick={() => loadFile(f)}
                >
                  {label || f}
                </div>
              );
            })}

            {otherFiles.length > 0 && (
              <>
                <div className="text-xs text-muted-foreground uppercase tracking-wider mt-4 mb-2">Other files</div>
                {otherFiles.map((f) => (
                  <div
                    key={f}
                    className={`text-xs px-2 py-1 cursor-pointer rounded transition-colors ${
                      selectedFile === f ? "bg-accent text-accent-foreground" : "hover:bg-muted/50 text-muted-foreground"
                    }`}
                    onClick={() => loadFile(f)}
                    title="Read-only"
                  >
                    {f}
                  </div>
                ))}
              </>
            )}
          </div>

          {/* File editor */}
          <div className="flex-1 flex flex-col min-w-0">
            {selectedFile ? (
              <>
                <div className="flex items-center justify-between border-b px-3 py-1">
                  <span className="text-xs text-muted-foreground font-mono">{selectedFile}</span>
                  <div className="flex items-center gap-2">
                    {isReadOnly && (
                      <Badge variant="outline" className="text-xs text-muted-foreground">read-only</Badge>
                    )}
                    {!isReadOnly && hasChanges && (
                      <Badge variant="secondary" className="text-xs">modified</Badge>
                    )}
                    {!isReadOnly && (
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={!editedContent.trim() || improving}
                        onClick={improveWithAI}
                      >
                        {improving ? "Improving..." : "Improve with AI"}
                      </Button>
                    )}
                    {!isReadOnly && (
                      <Button
                        size="sm"
                        disabled={!hasChanges || saving}
                        onClick={saveFile}
                      >
                        {saving ? "Saving..." : "Save"}
                      </Button>
                    )}
                    {selectedFile.endsWith("spec.md") && (
                      <>
                        <Button
                          size="sm"
                          onClick={() => setCopilotOpen(true)}
                        >
                          Chat about this spec
                        </Button>
                        <Button
                          size="sm"
                          onClick={() => {
                            const specName = selectedFile.replace("openspec/specs/", "").replace("/spec.md", "");
                            navigate(`/new?project=${name}&spec=${specName}`);
                          }}
                        >
                          Audit this spec
                        </Button>
                      </>
                    )}
                  </div>
                </div>
                <div className="flex-1 min-h-0">
                  <MarkdownEditor value={editedContent} onChange={setEditedContent} minHeight="100%" readOnly={isReadOnly} />
                </div>
              </>
            ) : (
              <div className="flex h-full items-center justify-center text-muted-foreground text-sm">
                Select a file to view
              </div>
            )}
          </div>
        </TabsContent>

        {/* Runs tab */}
        <TabsContent value="runs" className="flex flex-col flex-1 min-h-0 overflow-y-auto overscroll-none mt-0">
          {runs.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full gap-3 text-muted-foreground">
              <span className="text-sm">No runs yet</span>
              <Button size="sm" variant="ghost" asChild>
                <Link to={`/new?project=${name}`}>+ New Run</Link>
              </Button>
            </div>
          ) : (
            <div className="divide-y divide-border/50">
              {runs.map((run) => (
                <Link
                  key={run.id}
                  to={`/run/${run.id}`}
                  className="flex items-center gap-3 px-4 py-3 hover:bg-muted/30 transition-colors"
                >
                  <RunStatusBadge phase={run.status.phase} />
                  <span className="flex-1 min-w-0 text-sm truncate">
                    {run.spec.displayName || run.name}
                  </span>
                  <span className="text-xs text-muted-foreground shrink-0">
                    {formatAge(run.createdAt)}
                  </span>
                </Link>
              ))}
            </div>
          )}
        </TabsContent>

        {/* Settings tab */}
        <TabsContent value="settings" className="flex-1 overflow-y-auto overscroll-none mt-0">
          <div className="p-4 space-y-4 max-w-2xl">
            <div>
              <label className="text-xs text-muted-foreground block mb-1">Name</label>
              {renaming ? (
                <div className="flex gap-2">
                  <Input
                    className="h-8 text-sm font-mono flex-1"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, "-"))}
                    onKeyDown={(e) => { if (e.key === "Enter") handleRename(); if (e.key === "Escape") setRenaming(false); }}
                    autoFocus
                  />
                  <Button size="sm" onClick={handleRename} disabled={renameSaving || !newName.trim()}>
                    {renameSaving ? "Renaming..." : "Rename"}
                  </Button>
                  <Button size="sm" variant="ghost" onClick={() => setRenaming(false)}>Cancel</Button>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <span className="font-mono text-sm">{project.name}</span>
                  <Button size="sm" variant="ghost" className="text-xs text-muted-foreground h-6 px-2" onClick={() => { setNewName(project.name); setRenaming(true); }}>
                    Rename
                  </Button>
                </div>
              )}
            </div>

            <div>
              <label className="text-xs text-muted-foreground block mb-1">Description</label>
              <Input
                className="h-8 text-sm"
                value={editDescription}
                onChange={(e) => { setEditDescription(e.target.value); setSettingsDirty(true); }}
                placeholder="Project description"
              />
            </div>

            <div>
              <label className="text-xs text-muted-foreground block mb-1">Workspace</label>
              <p className="text-xs text-muted-foreground">Workspace configuration coming soon.</p>
            </div>

            {/* Devbox Packages */}
            <div>
              <label className="text-xs text-muted-foreground block mb-1">Devbox Packages</label>
              {editDevboxPackages.map((pkg) => (
                <div key={pkg} className="text-sm font-mono flex items-center gap-2 mb-1">
                  <span className="flex-1">{pkg}</span>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => removePackage(pkg)}
                  >
                    ×
                  </Button>
                </div>
              ))}
              <div className="flex gap-2 mt-1">
                <Input
                  className="h-8 text-sm flex-1"
                  placeholder="e.g. go@1.22"
                  value={newPackage}
                  onChange={(e) => setNewPackage(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter") addPackage(); }}
                />
                <Button size="sm" onClick={addPackage}>+</Button>
              </div>
            </div>

            {/* Run Defaults */}
            <div className="space-y-3">
              <label className="text-xs text-muted-foreground block">Run Defaults</label>

              <div className="grid grid-cols-3 gap-2">
                {(["modelTier", "manageModelTier", "implementModelTier"] as const).map((field) => {
                  const labels: Record<string, string> = {
                    modelTier: "Model",
                    manageModelTier: "Manage model",
                    implementModelTier: "Implement model",
                  };
                  return (
                    <div key={field}>
                      <label className="text-xs text-muted-foreground block mb-1">{labels[field]}</label>
                      <CustomSelect
                        value={editDefaults[field]}
                        onChange={(v) => { setEditDefaults({ ...editDefaults, [field]: v }); setSettingsDirty(true); }}
                        options={[{ value: "", label: "—" }, ...MODEL_OPTIONS.map((m) => ({ value: m.value, label: m.label }))]}
                        className="h-8 text-sm font-mono"
                      />
                    </div>
                  );
                })}
              </div>

              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="text-xs text-muted-foreground block mb-1">TTL (seconds)</label>
                  <Input
                    className="h-8 text-sm"
                    type="number"
                    value={editDefaults.ttlSeconds}
                    onChange={(e) => { setEditDefaults({ ...editDefaults, ttlSeconds: e.target.value }); setSettingsDirty(true); }}
                    placeholder="e.g. 3600"
                  />
                </div>

                <div>
                  <label className="text-xs text-muted-foreground block mb-1">Orchestration Mode</label>
                  <CustomSelect
                    value={editDefaults.orchestrationMode}
                    onChange={(v) => { setEditDefaults({ ...editDefaults, orchestrationMode: v }); setSettingsDirty(true); }}
                    options={[
                      { value: "", label: "—" },
                      { value: "spec-driven", label: "spec-driven" },
                      { value: "prompt-driven", label: "prompt-driven" },
                    ]}
                    className="h-8 text-sm"
                  />
                </div>
              </div>

              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <input
                    type="checkbox"
                    className="h-4 w-4 rounded border-input accent-primary"
                    checked={editDefaults.autoPush}
                    onChange={(e) => { setEditDefaults({ ...editDefaults, autoPush: e.target.checked }); setSettingsDirty(true); }}
                  />
                  Auto Push
                </label>
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <input
                    type="checkbox"
                    className="h-4 w-4 rounded border-input accent-primary"
                    checked={editDefaults.autoPR}
                    onChange={(e) => { setEditDefaults({ ...editDefaults, autoPR: e.target.checked }); setSettingsDirty(true); }}
                  />
                  Auto PR
                </label>
              </div>

              <div>
                <label className="text-xs text-muted-foreground block mb-1">PR Base Branch</label>
                <Input
                  className="h-8 text-sm"
                  value={editDefaults.prBaseBranch}
                  onChange={(e) => { setEditDefaults({ ...editDefaults, prBaseBranch: e.target.value }); setSettingsDirty(true); }}
                  placeholder="main"
                  disabled={!editDefaults.autoPR}
                />
              </div>
            </div>

            <div>
              <label className="text-xs text-muted-foreground block mb-1">Config Repo</label>
              <div className="text-sm font-mono bg-muted px-2 py-1 rounded">
                {project.configRepoURL || "Not ready"}
              </div>
            </div>

            {settingsDirty && (
              <div className="flex items-center gap-2 pt-2">
                <Button size="sm" onClick={saveSettings} disabled={savingSettings}>
                  {savingSettings ? "Saving..." : "Save Settings"}
                </Button>
                <Button size="sm" variant="ghost" onClick={() => {
                  setEditDescription(project.description || "");
                  setEditDevboxPackages(project.devbox?.packages || []);
                  setEditDefaults({
                    modelTier: project.defaults?.modelTier || "",
                    manageModelTier: project.defaults?.manageModelTier || "",
                    implementModelTier: project.defaults?.implementModelTier || "",
                    ttlSeconds: project.defaults?.ttlSeconds != null ? String(project.defaults.ttlSeconds) : "",
                    orchestrationMode: project.defaults?.orchestrationMode || "",
                    autoPush: project.defaults?.autoPush || false,
                    autoPR: project.defaults?.autoPR || false,
                    prBaseBranch: project.defaults?.prBaseBranch || "",
                  });
                  setSettingsDirty(false);
                }}>
                  Discard
                </Button>
              </div>
            )}
          </div>
        </TabsContent>
      </Tabs>

    </div>
  );
}
