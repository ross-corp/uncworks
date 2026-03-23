import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { apiFetch } from "../hooks/apiFetch";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import MarkdownEditor from "../components/MarkdownEditor";

interface ProjectDetail {
  name: string;
  displayName: string;
  description: string;
  repos: { url: string; branch: string }[];
  devbox?: { packages: string[] };
  defaults?: Record<string, unknown>;
  configRepoReady: boolean;
  configRepoURL: string;
  runCount: number;
  totalCost: string;
}

export default function ProjectDetailView() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [project, setProject] = useState<ProjectDetail | null>(null);
  const [files, setFiles] = useState<string[]>([]);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState("");
  const [editedContent, setEditedContent] = useState("");
  const [saving, setSaving] = useState(false);
  const [improving, setImproving] = useState(false);
  const [tab, setTab] = useState<"specs" | "settings">("specs");

  // Settings editing state
  const [editRepos, setEditRepos] = useState<{ url: string; branch: string }[]>([]);
  const [editDisplayName, setEditDisplayName] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [settingsDirty, setSettingsDirty] = useState(false);
  const [savingSettings, setSavingSettings] = useState(false);

  // New spec creation
  const [showNewSpec, setShowNewSpec] = useState(false);
  const [newSpecName, setNewSpecName] = useState("");
  const [creatingSpec, setCreatingSpec] = useState(false);

  const fetchProject = useCallback(async () => {
    if (!name) return;
    try {
      const resp = await apiFetch(`/api/v1/projects/${name}`);
      if (resp.ok) {
        const data = await resp.json();
        setProject(data);
        setEditRepos(data.repos || []);
        setEditDisplayName(data.displayName || "");
        setEditDescription(data.description || "");
        setSettingsDirty(false);
      }
    } catch { /* silent */ }
  }, [name]);

  const fetchFiles = useCallback(async () => {
    if (!name) return;
    try {
      const resp = await apiFetch(`/api/v1/projects/${name}/files`);
      if (resp.ok) setFiles(await resp.json());
    } catch { /* silent */ }
  }, [name]);

  useEffect(() => {
    fetchProject();
    fetchFiles();
  }, [fetchProject, fetchFiles]);

  async function loadFile(path: string) {
    if (!name) return;
    setSelectedFile(path);
    try {
      const resp = await apiFetch(`/api/v1/projects/${name}/files/${path}`);
      if (resp.ok) {
        const data = await resp.json();
        setFileContent(data.content);
        setEditedContent(data.content);
      }
    } catch { /* silent */ }
  }

  async function saveFile() {
    if (!name || !selectedFile) return;
    setSaving(true);
    try {
      await apiFetch(`/api/v1/projects/${name}/files/${selectedFile}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ content: editedContent, commitMessage: `update ${selectedFile}` }),
      });
      setFileContent(editedContent);
    } catch { /* silent */ }
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
    } catch { /* silent */ }
    setImproving(false);
  }

  async function createSpec() {
    if (!name || !newSpecName.trim()) return;
    const specSlug = newSpecName.trim().toLowerCase().replace(/[^a-z0-9-]/g, "-");
    const path = `openspec/specs/${specSlug}/spec.md`;
    setCreatingSpec(true);
    try {
      await apiFetch(`/api/v1/projects/${name}/files/${path}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          content: `# ${newSpecName.trim()}\n\n## Requirements\n\n- \n\n## Acceptance Criteria\n\n- \n`,
          commitMessage: `create spec: ${specSlug}`,
        }),
      });
      setShowNewSpec(false);
      setNewSpecName("");
      await fetchFiles();
      loadFile(path);
    } catch { /* silent */ }
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
          displayName: editDisplayName,
          description: editDescription,
          repos: editRepos.filter((r) => r.url.trim()),
        }),
      });
      if (resp.ok) {
        setProject(await resp.json());
        setSettingsDirty(false);
      }
    } catch { /* silent */ }
    setSavingSettings(false);
  }

  function updateRepo(i: number, field: "url" | "branch", value: string) {
    setEditRepos(editRepos.map((r, idx) => idx === i ? { ...r, [field]: value } : r));
    setSettingsDirty(true);
  }
  function addRepo() {
    setEditRepos([...editRepos, { url: "", branch: "main" }]);
    setSettingsDirty(true);
  }
  function removeRepo(i: number) {
    setEditRepos(editRepos.filter((_, idx) => idx !== i));
    setSettingsDirty(true);
  }

  const specFiles = files.filter((f) => f.startsWith("openspec/specs/"));
  const hasChanges = editedContent !== fileContent;

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
      <div className="flex items-center justify-between border-b px-4 py-2">
        <div className="flex items-center gap-3">
          <Button size="sm" variant="ghost" className="h-6 text-[11px]" onClick={() => navigate("/projects")}>
            &larr; Projects
          </Button>
          <span className="font-semibold">{project.displayName || project.name}</span>
          {project.configRepoReady ? (
            <Badge variant="outline" className="text-[10px] border-green-500/40 text-green-500">ready</Badge>
          ) : (
            <Badge variant="secondary" className="text-[10px]">provisioning</Badge>
          )}
          <span className="text-xs text-muted-foreground">{project.runCount} runs</span>
          {project.totalCost && (
            <span className="text-xs text-muted-foreground">{project.totalCost}</span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" className="h-6 text-[11px]" onClick={() => navigate(`/new?project=${name}`)}>
            + new run
          </Button>
        </div>
      </div>

      {/* Tab bar */}
      <div className="flex items-center gap-1 border-b px-4 py-1">
        <Badge variant={tab === "specs" ? "default" : "outline"} className="cursor-pointer text-[11px]" onClick={() => setTab("specs")}>
          Specs
        </Badge>
        <Badge variant={tab === "settings" ? "default" : "outline"} className="cursor-pointer text-[11px]" onClick={() => setTab("settings")}>
          Settings
        </Badge>
      </div>

      {/* Content */}
      <div className="flex-1 min-h-0 flex">
        {tab === "specs" && (
          <>
            {/* Spec file tree */}
            <div className="w-56 border-r overflow-y-auto p-2">
              <div className="flex items-center justify-between mb-2">
                <span className="text-[10px] text-muted-foreground uppercase tracking-wider">Specs</span>
                <button
                  onClick={() => setShowNewSpec(!showNewSpec)}
                  className="text-[10px] text-muted-foreground hover:text-foreground"
                >
                  + new
                </button>
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
                    <Button size="sm" className="h-5 text-[10px] flex-1" onClick={createSpec} disabled={creatingSpec || !newSpecName.trim()}>
                      {creatingSpec ? "..." : "Create"}
                    </Button>
                    <Button size="sm" variant="ghost" className="h-5 text-[10px]" onClick={() => setShowNewSpec(false)}>
                      Cancel
                    </Button>
                  </div>
                </div>
              )}

              {specFiles.length === 0 && !showNewSpec && (
                <div className="text-xs text-muted-foreground p-2">No specs yet</div>
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

              <div className="text-[10px] text-muted-foreground uppercase tracking-wider mt-4 mb-2">Other files</div>
              {files.filter((f) => !f.startsWith("openspec/specs/")).map((f) => (
                <div
                  key={f}
                  className={`text-xs px-2 py-1 cursor-pointer rounded transition-colors ${
                    selectedFile === f ? "bg-accent text-accent-foreground" : "hover:bg-muted/50 text-muted-foreground"
                  }`}
                  onClick={() => loadFile(f)}
                >
                  {f}
                </div>
              ))}
            </div>

            {/* File editor */}
            <div className="flex-1 flex flex-col min-w-0">
              {selectedFile ? (
                <>
                  <div className="flex items-center justify-between border-b px-3 py-1">
                    <span className="text-xs text-muted-foreground font-mono">{selectedFile}</span>
                    <div className="flex items-center gap-2">
                      {hasChanges && (
                        <Badge variant="secondary" className="text-[10px]">modified</Badge>
                      )}
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-6 text-[11px]"
                        disabled={!editedContent.trim() || improving}
                        onClick={improveWithAI}
                      >
                        {improving ? "Improving..." : "Improve with AI"}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        className="h-6 text-[11px]"
                        disabled={!hasChanges || saving}
                        onClick={saveFile}
                      >
                        {saving ? "Saving..." : "Save"}
                      </Button>
                      {selectedFile.endsWith("spec.md") && (
                        <Button
                          size="sm"
                          className="h-6 text-[11px]"
                          onClick={() => {
                            const specName = selectedFile.replace("openspec/specs/", "").replace("/spec.md", "");
                            navigate(`/new?project=${name}&spec=${specName}`);
                          }}
                        >
                          Run this spec
                        </Button>
                      )}
                    </div>
                  </div>
                  <div className="flex-1 min-h-0">
                    <MarkdownEditor value={editedContent} onChange={setEditedContent} minHeight="100%" />
                  </div>
                </>
              ) : (
                <div className="flex h-full items-center justify-center text-muted-foreground text-sm">
                  Select a file to view or create a new spec
                </div>
              )}
            </div>
          </>
        )}

        {tab === "settings" && (
          <div className="flex-1 p-4 space-y-4 max-w-2xl overflow-y-auto">
            <div>
              <label className="text-xs text-muted-foreground block mb-1">Display Name</label>
              <Input
                className="h-8 text-sm"
                value={editDisplayName}
                onChange={(e) => { setEditDisplayName(e.target.value); setSettingsDirty(true); }}
              />
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
              <label className="text-xs text-muted-foreground block mb-1">Repositories</label>
              {editRepos.map((r, i) => (
                <div key={i} className="flex gap-2 mb-1">
                  <Input
                    className="flex-1 h-8 text-sm"
                    value={r.url}
                    onChange={(e) => updateRepo(i, "url", e.target.value)}
                    placeholder="https://github.com/org/repo"
                  />
                  <Input
                    className="w-20 h-8 text-sm"
                    value={r.branch}
                    onChange={(e) => updateRepo(i, "branch", e.target.value)}
                    placeholder="main"
                  />
                  <Button size="sm" variant="ghost" className="h-8 px-2 text-muted-foreground" onClick={() => removeRepo(i)}>
                    x
                  </Button>
                </div>
              ))}
              <Button size="sm" variant="ghost" className="h-6 text-[11px] text-muted-foreground" onClick={addRepo}>
                + add repository
              </Button>
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
                  setEditRepos(project.repos || []);
                  setEditDisplayName(project.displayName || "");
                  setEditDescription(project.description || "");
                  setSettingsDirty(false);
                }}>
                  Discard
                </Button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
