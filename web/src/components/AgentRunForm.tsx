import { useState, useEffect } from "react";
import type { Backend, ModelTier, Repository, PipelineConfig } from "../types/agent-run";
import { BACKEND_OPTIONS, MODEL_TIER_OPTIONS } from "../types/agent-run";
import type { Workspace } from "../hooks/useWorkspaces";
import { useGitHub } from "../hooks/useGitHub";
import GitHubModal from "./GitHubModal";
import SpecEditor from "./SpecEditor";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

type InputMode = "prompt" | "spec";

export default function AgentRunForm({
  repos,
  workspaces,
  cloneSource,
  onSubmit,
  onCancel,
}: {
  repos: string[];
  workspaces: Workspace[];
  cloneSource?: { name: string; spec: { repos: Repository[]; prompt: string; backend: Backend; modelTier: ModelTier; ttlSeconds: number; specContent?: string; specSource?: string; workspaceName?: string } };
  onSubmit: (data: {
    name: string;
    repos: Repository[];
    workspaceName?: string;
    prompt: string;
    backend: Backend;
    modelTier: ModelTier;
    ttlSeconds: number;
    specContent?: string;
    specSource?: string;
    orchestrationMode?: string;
    pipelineConfig?: PipelineConfig;
  }) => void;
  onCancel: () => void;
}) {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onCancel();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onCancel]);

  const { pullSpec, pushSpec } = useGitHub();
  const [gitHubModal, setGitHubModal] = useState<"load" | "push" | null>(null);
  const [githubError, setGithubError] = useState<string | null>(null);
  const [specSource, setSpecSource] = useState<string>("editor");
  const [name, setName] = useState(cloneSource ? `${cloneSource.name}-clone` : "");
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string | null>(null);
  const [runRepos, setRunRepos] = useState<Repository[]>(
    cloneSource?.spec.repos?.length
      ? cloneSource.spec.repos.map((r) => ({ ...r }))
      : [{ url: repos[0] ?? "", branch: "main" }]
  );
  const [prompt, setPrompt] = useState(cloneSource?.spec.prompt ?? "");
  const [specContent, setSpecContent] = useState(cloneSource?.spec.specContent ?? "");
  const [inputMode, setInputMode] = useState<InputMode>(cloneSource?.spec.specContent ? "spec" : "prompt");
  const [backend, setBackend] = useState<Backend>(cloneSource?.spec.backend ?? "pod");
  const [modelTier, setModelTier] = useState<ModelTier>(cloneSource?.spec.modelTier ?? "default-cloud");
  const [ttlSeconds, setTtlSeconds] = useState(cloneSource?.spec.ttlSeconds ?? 3600);
  const [useSpecDriven, setUseSpecDriven] = useState(false);
  const [pipelineExpanded, setPipelineExpanded] = useState(false);
  const [planModel, setPlanModel] = useState("default-cloud");
  const [execModel, setExecModel] = useState("default-cloud");
  const [verifyModel, setVerifyModel] = useState("default-cloud");

  function selectWorkspace(id: string | null) {
    setSelectedWorkspaceId(id);
    if (id) {
      const ws = workspaces.find((w) => w.id === id);
      if (ws) {
        setRunRepos(ws.repos.length > 0 ? ws.repos.map((r) => ({ ...r })) : [{ url: "", branch: "main" }]);
      }
    }
  }

  function updateRepo(index: number, field: keyof Repository, value: string) {
    setRunRepos((prev) => prev.map((r, i) => (i === index ? { ...r, [field]: value } : r)));
  }

  function removeRepo(index: number) {
    setRunRepos((prev) => (prev.length <= 1 ? prev : prev.filter((_, i) => i !== index)));
  }

  function addRepo() {
    setRunRepos((prev) => [...prev, { url: "", branch: "main" }]);
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const validRepos = runRepos.filter((r) => r.url.trim());
    if (!name.trim() || validRepos.length === 0) return;
    if (inputMode === "prompt" && !prompt.trim()) return;
    if (inputMode === "spec" && !specContent.trim()) return;

    const ws = selectedWorkspaceId ? workspaces.find((w) => w.id === selectedWorkspaceId) : null;

    onSubmit({
      name: name.trim(),
      repos: validRepos.map((r) => ({ url: r.url.trim(), branch: r.branch.trim() || "main" })),
      workspaceName: ws?.name,
      prompt: inputMode === "prompt" ? prompt.trim() : "",
      backend,
      modelTier,
      ttlSeconds,
      specContent: inputMode === "spec" ? specContent : undefined,
      specSource: inputMode === "spec" ? specSource : undefined,
      orchestrationMode: useSpecDriven ? "spec-driven" : undefined,
      pipelineConfig: useSpecDriven ? {
        plan: { model: planModel },
        execute: { model: execModel },
        verify: { model: verifyModel },
      } : undefined,
    });
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-background/80 backdrop-blur-sm pt-[5vh]"
      onClick={(e) => {
        if (e.target === e.currentTarget) onCancel();
      }}
    >
      <form
        data-testid="form-modal"
        onSubmit={handleSubmit}
        className={`w-full border border-border bg-card shadow-2xl ${
          inputMode === "spec" ? "max-w-4xl" : "max-w-lg"
        }`}
      >
        <div className="flex items-center justify-between border-b border-border px-5 py-3">
          <h2 className="text-sm font-semibold fx-glow">New Agent Run</h2>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onCancel}
            aria-label="Close"
          >
            &times;
          </Button>
        </div>

        <div className="flex flex-col gap-4 p-5 max-h-[80vh] overflow-y-auto">
          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">
              Name
            </label>
            <Input
              data-testid="form-name-input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="fix-auth-middleware"
              autoFocus
            />
          </div>

          {/* Workspace Selector */}
          {workspaces.length > 0 && (
            <div>
              <label className="mb-1 block text-xs font-medium text-muted-foreground">
                Workspace
              </label>
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => selectWorkspace(null)}
                  className={`px-3 py-1 text-xs font-medium transition-colors ${
                    selectedWorkspaceId === null
                      ? "bg-muted text-foreground"
                      : "text-muted-foreground/60 hover:text-muted-foreground"
                  }`}
                >
                  Custom repos
                </button>
                {workspaces.map((ws) => (
                  <button
                    key={ws.id}
                    type="button"
                    data-testid={`form-workspace-${ws.name}`}
                    onClick={() => selectWorkspace(ws.id)}
                    className={`px-3 py-1 text-xs font-medium transition-colors ${
                      selectedWorkspaceId === ws.id
                        ? "bg-muted text-foreground"
                        : "text-muted-foreground/60 hover:text-muted-foreground"
                    }`}
                  >
                    {ws.name}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Multi-Repo List */}
          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">
              Repositories
            </label>
            <div className="space-y-2">
              {runRepos.map((repo, i) => (
                <div key={i} className="flex items-center gap-2">
                  <div className="flex-1">
                    <Input
                      data-testid={`form-repo-row-${i}-url`}
                      list="known-repos"
                      value={repo.url}
                      onChange={(e) => updateRepo(i, "url", e.target.value)}
                      placeholder="https://github.com/org/repo"
                    />
                  </div>
                  <div className="w-24">
                    <Input
                      data-testid={`form-repo-row-${i}-branch`}
                      value={repo.branch}
                      onChange={(e) => updateRepo(i, "branch", e.target.value)}
                      placeholder="main"
                    />
                  </div>
                  {runRepos.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => removeRepo(i)}
                      className="text-xs text-destructive"
                    >
                      &times;
                    </Button>
                  )}
                </div>
              ))}
            </div>
            <datalist id="known-repos">
              {repos.map((url) => (
                <option key={url} value={url} />
              ))}
            </datalist>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              data-testid="form-add-repo"
              onClick={addRepo}
              className="mt-2 text-xs"
            >
              + Add repo
            </Button>
          </div>

          <div className="flex gap-3">
            <div className="w-24">
              <label className="mb-1 block text-xs font-medium text-muted-foreground">
                TTL (sec)
              </label>
              <Input
                type="number"
                className="text-center"
                value={ttlSeconds}
                onChange={(e) => setTtlSeconds(Number(e.target.value))}
                min={300}
                max={86400}
              />
            </div>
            <div className="flex-1">
              <label className="mb-1 block text-xs font-medium text-muted-foreground">
                Backend
              </label>
              <select
                data-testid="form-backend-select"
                className="w-full border border-input bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors focus:border-primary"
                value={backend}
                onChange={(e) => setBackend(e.target.value as Backend)}
              >
                {BACKEND_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex-1">
              <label className="mb-1 block text-xs font-medium text-muted-foreground">
                Model
              </label>
              <select
                data-testid="form-model-select"
                className="w-full border border-input bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors focus:border-primary"
                value={modelTier}
                onChange={(e) => setModelTier(e.target.value as ModelTier)}
              >
                {MODEL_TIER_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label} · {opt.description}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Prompt/Spec Tab Selector */}
          <div>
            <div className="mb-2 flex gap-1">
              <button
                type="button"
                data-testid="form-tab-prompt"
                onClick={() => setInputMode("prompt")}
                className={`px-3 py-1 text-xs font-medium transition-colors ${
                  inputMode === "prompt"
                    ? "bg-muted text-foreground"
                    : "text-muted-foreground/60 hover:text-muted-foreground"
                }`}
              >
                Prompt
              </button>
              <button
                type="button"
                data-testid="form-tab-spec"
                onClick={() => setInputMode("spec")}
                className={`px-3 py-1 text-xs font-medium transition-colors ${
                  inputMode === "spec"
                    ? "bg-muted text-foreground"
                    : "text-muted-foreground/60 hover:text-muted-foreground"
                }`}
              >
                Spec
              </button>
            </div>

            {inputMode === "prompt" ? (
              <textarea
                data-testid="form-prompt-input"
                className="w-full border border-input bg-background px-3 py-2 text-sm text-foreground placeholder-muted-foreground/60 outline-none transition-colors focus:border-primary min-h-[120px] resize-y"
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder="Describe what the agent should do..."
              />
            ) : (
              <div>
                <SpecEditor
                  value={specContent}
                  onChange={(v) => {
                    setSpecContent(v);
                    if (!specSource.startsWith("github:")) {
                      setSpecSource("editor");
                    }
                  }}
                  height="400px"
                />
                <div className="mt-2 flex gap-2">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => setGitHubModal("load")}
                    className="text-xs"
                  >
                    Load from GitHub
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => setGitHubModal("push")}
                    className="text-xs"
                  >
                    Push to GitHub
                  </Button>
                </div>
                {githubError && (
                  <p className="mt-1 text-xs text-destructive">{githubError}</p>
                )}
              </div>
            )}
          </div>

          {/* Spec-Driven Pipeline Toggle */}
          <div>
            <label className="flex items-center gap-2 text-xs font-medium text-muted-foreground cursor-pointer">
              <input
                type="checkbox"
                checked={useSpecDriven}
                onChange={(e) => setUseSpecDriven(e.target.checked)}
                className="accent-primary"
              />
              Spec-driven mode (Plan → Execute → Verify)
            </label>

            {useSpecDriven && (
              <div className="mt-2">
                <button
                  type="button"
                  onClick={() => setPipelineExpanded(!pipelineExpanded)}
                  className="text-xs text-muted-foreground/60 hover:text-muted-foreground"
                >
                  {pipelineExpanded ? "▼" : "▶"} Pipeline settings
                </button>
                {pipelineExpanded && (
                  <div className="mt-2 space-y-2 pl-4 border-l border-border">
                    {(["plan", "execute", "verify"] as const).map((stage) => {
                      const model = stage === "plan" ? planModel : stage === "execute" ? execModel : verifyModel;
                      const setModel = stage === "plan" ? setPlanModel : stage === "execute" ? setExecModel : setVerifyModel;
                      return (
                        <div key={stage} className="flex items-center gap-2">
                          <span className="text-xs font-medium text-muted-foreground w-16 capitalize">{stage}</span>
                          <select
                            className="flex-1 border border-input bg-background px-2 py-1 text-xs text-foreground outline-none"
                            value={model}
                            onChange={(e) => setModel(e.target.value)}
                          >
                            <option value="default-cloud">Default (qwen3-coder)</option>
                            <option value="qwen3-coder">qwen3-coder</option>
                            <option value="mistral-small">mistral-small</option>
                            <option value="gemma-3-4b">gemma-3-4b</option>
                            <option value="qwen2.5:0.5b">qwen2.5:0.5b (CI only)</option>
                          </select>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        <div className="flex justify-end gap-2 border-t border-border px-5 py-3">

          <Button type="button" variant="ghost" onClick={onCancel}>
            Cancel
          </Button>
          <Button data-testid="form-submit" type="submit">
            Create Run
          </Button>
        </div>
      </form>

      {gitHubModal === "load" && (
        <GitHubModal
          mode="load"
          onLoad={async (repo, path) => {
            try {
              const { content } = await pullSpec(repo, path);
              setSpecContent(content);
              setSpecSource(`github:${repo}/${path}`);
              setGithubError(null);
              setGitHubModal(null);
            } catch (err) {
              console.error("Failed to load spec from GitHub:", err);
              setGithubError(`Failed to load spec: ${err instanceof Error ? err.message : String(err)}`);
            }
          }}
          onClose={() => setGitHubModal(null)}
        />
      )}

      {gitHubModal === "push" && (
        <GitHubModal
          mode="push"
          onPush={async (repo, path, message) => {
            try {
              await pushSpec(repo, path, specContent, message);
              setSpecSource(`github:${repo}/${path}`);
              setGithubError(null);
              setGitHubModal(null);
            } catch (err) {
              console.error("Failed to push spec to GitHub:", err);
              setGithubError(`Failed to push spec: ${err instanceof Error ? err.message : String(err)}`);
            }
          }}
          onClose={() => setGitHubModal(null)}
        />
      )}
    </div>
  );
}
