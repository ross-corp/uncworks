import { useState, useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useClient, mapRun } from "../hooks/useClient";
import { useToast } from "../components/Toast";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import type { OrchestrationMode } from "../types/agent-run";

export default function NewRunView() {
  const client = useClient();
  const navigate = useNavigate();
  const { toast } = useToast();

  const [searchParams] = useSearchParams();
  const [prompt, setPrompt] = useState("");
  const [repo, setRepo] = useState("https://github.com/roshbhatia/neph.nvim");
  const [branch, setBranch] = useState("main");
  const [submitting, setSubmitting] = useState(false);
  const [mode, setMode] = useState<"prompt" | "spec">("prompt");
  const [specContent, setSpecContent] = useState("");

  // Clone support: pre-fill from cloned run
  useEffect(() => {
    const cloneId = searchParams.get("clone");
    if (!cloneId) return;
    client.getAgentRun(cloneId).then((raw) => {
      const run = mapRun(raw);
      setPrompt(run.spec.prompt || "");
      if (run.spec.repos?.[0]) {
        setRepo(run.spec.repos[0].url);
        setBranch(run.spec.repos[0].branch);
      }
      if (run.spec.specContent) {
        setSpecContent(run.spec.specContent);
        setMode("spec");
      }
    }).catch(() => {});
  }, [searchParams, client]);

  async function handleRun() {
    if (!prompt.trim() || !repo.trim()) return;
    setSubmitting(true);
    try {
      const run = await client.createAgentRun({
        backend: "Pod",
        repos: [{ url: repo.trim(), branch: branch.trim() || "main" }],
        prompt: prompt.trim(),
        ttlSeconds: 900,
        modelTier: "default",
        ...(mode === "spec" && specContent.trim()
          ? { specContent: specContent.trim(), orchestrationMode: "spec-driven" as OrchestrationMode }
          : {}),
      });
      toast("Run created", "success");
      navigate(`/run/${run.id}`);
    } catch (err) {
      toast("Failed to create run", "error");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-4 py-2">
        <span className="font-semibold">New Run</span>
        <span className="text-xs text-muted-foreground">esc cancel</span>
      </div>

      {/* Form */}
      <div className="mx-auto flex w-full max-w-2xl flex-1 flex-col gap-4 p-6">
        {/* Repo */}
        <div>
          <label className="mb-1 block text-xs text-muted-foreground">Repository</label>
          <div className="flex gap-2">
            <Input
              className="flex-1"
              value={repo}
              onChange={(e) => setRepo(e.target.value)}
              placeholder="https://github.com/org/repo"
            />
            <Input
              className="w-24"
              value={branch}
              onChange={(e) => setBranch(e.target.value)}
              placeholder="main"
            />
          </div>
        </div>

        {/* Mode tabs */}
        <div className="flex gap-1">
          <button
            className={`px-3 py-1 text-xs border ${mode === "prompt" ? "bg-accent text-accent-foreground" : "text-muted-foreground"}`}
            onClick={() => setMode("prompt")}
          >
            Prompt
          </button>
          <button
            className={`px-3 py-1 text-xs border ${mode === "spec" ? "bg-accent text-accent-foreground" : "text-muted-foreground"}`}
            onClick={() => setMode("spec")}
          >
            Spec
          </button>
        </div>

        {/* Prompt */}
        <div className="flex-1">
          <label className="mb-1 block text-xs text-muted-foreground">
            {mode === "prompt" ? "Prompt" : "Prompt (still required for spec runs)"}
          </label>
          <textarea
            autoFocus
            className={`w-full resize-none border bg-background p-3 text-sm outline-none focus:border-primary ${mode === "spec" ? "min-h-[80px]" : "h-full min-h-[200px]"}`}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="What should the agent do?"
            onKeyDown={(e) => {
              if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) handleRun();
            }}
          />
        </div>

        {/* Spec textarea (visible in spec mode) */}
        {mode === "spec" && (
          <div className="flex-1">
            <label className="mb-1 block text-xs text-muted-foreground">Spec</label>
            <textarea
              className="h-full min-h-[300px] w-full resize-none border bg-background p-3 text-sm font-mono outline-none focus:border-primary"
              value={specContent}
              onChange={(e) => setSpecContent(e.target.value)}
              placeholder="Paste your spec (markdown)..."
              onKeyDown={(e) => {
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) handleRun();
              }}
            />
          </div>
        )}

        {/* Config summary + actions */}
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted-foreground">
            qwen3:8b · 15m · single
          </span>
          <div className="flex gap-2">
            <Button variant="ghost" onClick={() => navigate("/")}>
              Cancel
            </Button>
            <Button onClick={handleRun} disabled={submitting || !prompt.trim()}>
              {submitting ? "Creating..." : "Run"}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
