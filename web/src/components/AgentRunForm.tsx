import { useState } from "react";
import type { Backend, ModelTier, Repository } from "../types/agent-run";
import { BACKEND_OPTIONS, MODEL_TIER_OPTIONS } from "../types/agent-run";
import type { Workspace } from "../hooks/useWorkspaces";
import { useGitHub } from "../hooks/useGitHub";
import GitHubModal from "./GitHubModal";
import SpecEditor from "./SpecEditor";

type InputMode = "prompt" | "spec";

export default function AgentRunForm({
  repos,
  workspaces,
  onSubmit,
  onCancel,
}: {
  repos: string[];
  workspaces: Workspace[];
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
  }) => void;
  onCancel: () => void;
}) {
  const { pullSpec, pushSpec } = useGitHub();
  const [gitHubModal, setGitHubModal] = useState<"load" | "push" | null>(null);
  const [specSource, setSpecSource] = useState<string>("editor");
  const [name, setName] = useState("");
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string | null>(null);
  const [runRepos, setRunRepos] = useState<Repository[]>([
    { url: repos[0] ?? "", branch: "main" },
  ]);
  const [prompt, setPrompt] = useState("");
  const [specContent, setSpecContent] = useState("");
  const [inputMode, setInputMode] = useState<InputMode>("prompt");
  const [backend, setBackend] = useState<Backend>("pod");
  const [modelTier, setModelTier] = useState<ModelTier>("default-cloud");
  const [ttlSeconds, setTtlSeconds] = useState(3600);

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
    });
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/60 pt-[10vh]"
      onClick={(e) => {
        if (e.target === e.currentTarget) onCancel();
      }}
    >
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-lg rounded-lg border border-edge bg-surface-1 shadow-2xl"
      >
        <div className="flex items-center justify-between border-b border-edge px-5 py-3">
          <h2 className="text-sm font-semibold">New Agent Run</h2>
          <button
            type="button"
            onClick={onCancel}
            className="btn-ghost px-2"
          >
            &times;
          </button>
        </div>

        <div className="flex max-h-[70vh] flex-col gap-4 overflow-y-auto p-5">
          <div>
            <label className="mb-1 block text-xs font-medium text-txt-secondary">
              Name
            </label>
            <input
              className="input-field"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="fix-auth-middleware"
              autoFocus
            />
          </div>

          {/* Workspace Selector */}
          {workspaces.length > 0 && (
            <div>
              <label className="mb-1 block text-xs font-medium text-txt-secondary">
                Workspace
              </label>
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => selectWorkspace(null)}
                  className={`rounded px-3 py-1 text-xs font-medium transition-colors ${
                    selectedWorkspaceId === null
                      ? "bg-surface-2 text-txt-primary"
                      : "text-txt-tertiary hover:text-txt-secondary"
                  }`}
                >
                  Custom repos
                </button>
                {workspaces.map((ws) => (
                  <button
                    key={ws.id}
                    type="button"
                    onClick={() => selectWorkspace(ws.id)}
                    className={`rounded px-3 py-1 text-xs font-medium transition-colors ${
                      selectedWorkspaceId === ws.id
                        ? "bg-surface-2 text-txt-primary"
                        : "text-txt-tertiary hover:text-txt-secondary"
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
            <label className="mb-1 block text-xs font-medium text-txt-secondary">
              Repositories
            </label>
            <div className="space-y-2">
              {runRepos.map((repo, i) => (
                <div key={i} className="flex items-center gap-2">
                  <div className="flex-1">
                    <input
                      list="known-repos"
                      className="input-field"
                      value={repo.url}
                      onChange={(e) => updateRepo(i, "url", e.target.value)}
                      placeholder="https://github.com/org/repo"
                    />
                  </div>
                  <div className="w-24">
                    <input
                      className="input-field"
                      value={repo.branch}
                      onChange={(e) => updateRepo(i, "branch", e.target.value)}
                      placeholder="main"
                    />
                  </div>
                  {runRepos.length > 1 && (
                    <button
                      type="button"
                      onClick={() => removeRepo(i)}
                      className="btn-ghost px-2 text-xs text-danger"
                    >
                      &times;
                    </button>
                  )}
                </div>
              ))}
            </div>
            <datalist id="known-repos">
              {repos.map((url) => (
                <option key={url} value={url} />
              ))}
            </datalist>
            <button
              type="button"
              onClick={addRepo}
              className="btn-ghost mt-2 text-xs"
            >
              + Add repo
            </button>
          </div>

          <div className="flex gap-3">
            <div className="w-24">
              <label className="mb-1 block text-xs font-medium text-txt-secondary">
                TTL (sec)
              </label>
              <input
                type="number"
                className="input-field text-center"
                value={ttlSeconds}
                onChange={(e) => setTtlSeconds(Number(e.target.value))}
                min={300}
                max={86400}
              />
            </div>
            <div className="flex-1">
              <label className="mb-1 block text-xs font-medium text-txt-secondary">
                Backend
              </label>
              <select
                className="input-field"
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
              <label className="mb-1 block text-xs font-medium text-txt-secondary">
                Model Tier
              </label>
              <select
                className="input-field"
                value={modelTier}
                onChange={(e) => setModelTier(e.target.value as ModelTier)}
              >
                {MODEL_TIER_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
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
                onClick={() => setInputMode("prompt")}
                className={`rounded px-3 py-1 text-xs font-medium transition-colors ${
                  inputMode === "prompt"
                    ? "bg-surface-2 text-txt-primary"
                    : "text-txt-tertiary hover:text-txt-secondary"
                }`}
              >
                Prompt
              </button>
              <button
                type="button"
                onClick={() => setInputMode("spec")}
                className={`rounded px-3 py-1 text-xs font-medium transition-colors ${
                  inputMode === "spec"
                    ? "bg-surface-2 text-txt-primary"
                    : "text-txt-tertiary hover:text-txt-secondary"
                }`}
              >
                Spec
              </button>
            </div>

            {inputMode === "prompt" ? (
              <textarea
                className="input-field min-h-[120px] resize-y"
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder="Describe what the agent should do..."
              />
            ) : (
              <>
                <SpecEditor
                  value={specContent}
                  onChange={(v) => {
                    setSpecContent(v);
                    if (!specSource.startsWith("github:")) {
                      setSpecSource("editor");
                    }
                  }}
                  height="200px"
                />
                <div className="mt-2 flex gap-2">
                  <button
                    type="button"
                    onClick={() => setGitHubModal("load")}
                    className="btn-ghost text-xs"
                  >
                    Load from GitHub
                  </button>
                  <button
                    type="button"
                    onClick={() => setGitHubModal("push")}
                    className="btn-ghost text-xs"
                  >
                    Push to GitHub
                  </button>
                </div>
              </>
            )}
          </div>
        </div>

        <div className="flex justify-end gap-2 border-t border-edge px-5 py-3">
          <button type="button" onClick={onCancel} className="btn-ghost">
            Cancel
          </button>
          <button type="submit" className="btn-primary">
            Create Run
          </button>
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
              setGitHubModal(null);
            } catch (err) {
              console.error("Failed to load spec from GitHub:", err);
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
              setGitHubModal(null);
            } catch (err) {
              console.error("Failed to push spec to GitHub:", err);
            }
          }}
          onClose={() => setGitHubModal(null)}
        />
      )}
    </div>
  );
}
