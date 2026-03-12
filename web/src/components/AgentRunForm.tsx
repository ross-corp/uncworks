import { useState } from "react";
import type { Backend, ModelTier } from "../types/agent-run";
import { BACKEND_OPTIONS, MODEL_TIER_OPTIONS } from "../types/agent-run";

export default function AgentRunForm({
  repos,
  onSubmit,
  onCancel,
}: {
  repos: { name: string; url: string }[];
  onSubmit: (data: {
    name: string;
    repoURL: string;
    branch: string;
    prompt: string;
    backend: Backend;
    modelTier: ModelTier;
    ttlSeconds: number;
  }) => void;
  onCancel: () => void;
}) {
  const [name, setName] = useState("");
  const [repoURL, setRepoURL] = useState(repos[0]?.url ?? "");
  const [branch, setBranch] = useState("main");
  const [prompt, setPrompt] = useState("");
  const [backend, setBackend] = useState<Backend>("pod");
  const [modelTier, setModelTier] = useState<ModelTier>("default-cloud");
  const [ttlSeconds, setTtlSeconds] = useState(3600);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim() || !repoURL || !prompt.trim()) return;
    onSubmit({
      name: name.trim(),
      repoURL,
      branch: branch.trim() || "main",
      prompt: prompt.trim(),
      backend,
      modelTier,
      ttlSeconds,
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

        <div className="flex flex-col gap-4 p-5">
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

          <div>
            <label className="mb-1 block text-xs font-medium text-txt-secondary">
              Repository
            </label>
            <select
              className="input-field"
              value={repoURL}
              onChange={(e) => setRepoURL(e.target.value)}
            >
              {repos.map((r) => (
                <option key={r.url} value={r.url}>
                  {r.name}
                </option>
              ))}
            </select>
          </div>

          <div className="flex gap-3">
            <div className="flex-1">
              <label className="mb-1 block text-xs font-medium text-txt-secondary">
                Branch
              </label>
              <input
                className="input-field"
                value={branch}
                onChange={(e) => setBranch(e.target.value)}
                placeholder="main"
              />
            </div>
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
          </div>

          <div className="flex gap-3">
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

          <div>
            <label className="mb-1 block text-xs font-medium text-txt-secondary">
              Prompt
            </label>
            <textarea
              className="input-field min-h-[120px] resize-y"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Describe what the agent should do..."
            />
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
    </div>
  );
}
