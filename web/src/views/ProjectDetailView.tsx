import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { apiFetch } from "../hooks/apiFetch";
import { Button } from "../components/ui/button";
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
  const [tab, setTab] = useState<"specs" | "settings">("specs");

  const fetchProject = useCallback(async () => {
    if (!name) return;
    try {
      const resp = await apiFetch(`/api/v1/projects/${name}`);
      if (resp.ok) setProject(await resp.json());
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
        body: JSON.stringify({
          content: editedContent,
          commitMessage: `update ${selectedFile}`,
        }),
      });
      setFileContent(editedContent);
    } catch { /* silent */ }
    setSaving(false);
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
          <Button
            size="sm"
            variant="outline"
            className="h-6 text-[11px]"
            onClick={() => navigate(`/new?project=${name}`)}
          >
            + new run
          </Button>
        </div>
      </div>

      {/* Tab bar */}
      <div className="flex items-center gap-1 border-b px-4 py-1">
        <Badge
          variant={tab === "specs" ? "default" : "outline"}
          className="cursor-pointer text-[11px]"
          onClick={() => setTab("specs")}
        >
          Specs
        </Badge>
        <Badge
          variant={tab === "settings" ? "default" : "outline"}
          className="cursor-pointer text-[11px]"
          onClick={() => setTab("settings")}
        >
          Settings
        </Badge>
      </div>

      {/* Content */}
      <div className="flex-1 min-h-0 flex">
        {tab === "specs" && (
          <>
            {/* Spec file tree */}
            <div className="w-56 border-r overflow-y-auto p-2">
              <div className="text-[10px] text-muted-foreground uppercase tracking-wider mb-2">Specs</div>
              {specFiles.length === 0 && (
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
                    <MarkdownEditor
                      value={editedContent}
                      onChange={setEditedContent}
                      minHeight="100%"
                    />
                  </div>
                </>
              ) : (
                <div className="flex h-full items-center justify-center text-muted-foreground text-sm">
                  Select a file to view
                </div>
              )}
            </div>
          </>
        )}

        {tab === "settings" && (
          <div className="flex-1 p-4 space-y-4 max-w-2xl">
            <div>
              <label className="text-xs text-muted-foreground block mb-1">Repositories</label>
              {project.repos?.map((r, i) => (
                <div key={i} className="text-sm font-mono bg-muted px-2 py-1 rounded mb-1">
                  {r.url} ({r.branch})
                </div>
              ))}
              {(!project.repos || project.repos.length === 0) && (
                <div className="text-xs text-muted-foreground">No repositories configured</div>
              )}
            </div>

            <div>
              <label className="text-xs text-muted-foreground block mb-1">Devbox Packages</label>
              <div className="flex flex-wrap gap-1">
                {project.devbox?.packages?.map((pkg) => (
                  <Badge key={pkg} variant="secondary" className="text-[11px]">{pkg}</Badge>
                ))}
                {(!project.devbox?.packages || project.devbox.packages.length === 0) && (
                  <div className="text-xs text-muted-foreground">No packages</div>
                )}
              </div>
            </div>

            <div>
              <label className="text-xs text-muted-foreground block mb-1">Config Repo</label>
              <div className="text-sm font-mono bg-muted px-2 py-1 rounded">
                {project.configRepoURL || "Not ready"}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
