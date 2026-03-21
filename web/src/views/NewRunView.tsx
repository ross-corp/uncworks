import { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useClient, mapRun } from "../hooks/useClient";
import { apiFetch } from "../hooks/apiFetch";
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

  // Auto-classification fields
  const [project, setProject] = useState("");
  const [feature, setFeature] = useState("");
  const [tags, setTags] = useState("");
  const [classifying, setClassifying] = useState(false);

  // Track whether user has manually edited classification fields
  const userEditedProject = useRef(false);
  const userEditedFeature = useRef(false);
  const userEditedTags = useRef(false);

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
      if (run.spec.project) {
        setProject(run.spec.project);
        userEditedProject.current = true;
      }
      if (run.spec.feature) {
        setFeature(run.spec.feature);
        userEditedFeature.current = true;
      }
      if (run.spec.tags?.length) {
        setTags(run.spec.tags.join(", "));
        userEditedTags.current = true;
      }
    }).catch(() => {});
  }, [searchParams, client]);

  // Auto-classify prompt on blur
  const handlePromptBlur = useCallback(async () => {
    if (prompt.trim().length <= 10) return;
    // Don't classify if all fields already have user edits
    if (userEditedProject.current && userEditedFeature.current && userEditedTags.current) return;

    setClassifying(true);
    try {
      const resp = await apiFetch("/api/v1/classify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          prompt: prompt.trim(),
          repos: [{ url: repo.trim(), branch: branch.trim() || "main" }],
        }),
      });
      if (!resp.ok) return;
      const data = await resp.json() as { project?: string; feature?: string; tags?: string[] };
      if (data.project && !userEditedProject.current) setProject(data.project);
      if (data.feature && !userEditedFeature.current) setFeature(data.feature);
      if (data.tags?.length && !userEditedTags.current) setTags(data.tags.join(", "));
    } catch {
      // Classification is best-effort — silently ignore errors
    } finally {
      setClassifying(false);
    }
  }, [prompt, repo, branch]);

  async function handleRun() {
    if (!prompt.trim() || !repo.trim()) return;
    setSubmitting(true);
    try {
      const parsedTags = tags
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean);

      const run = await client.createAgentRun({
        backend: "pod",
        repos: [{ url: repo.trim(), branch: branch.trim() || "main" }],
        prompt: prompt.trim(),
        ttlSeconds: 900,
        modelTier: "default",
        ...(project.trim() ? { project: project.trim() } : {}),
        ...(feature.trim() ? { feature: feature.trim() } : {}),
        ...(parsedTags.length > 0 ? { tags: parsedTags } : {}),
        ...(mode === "spec" && specContent.trim()
          ? { specContent: specContent.trim(), orchestrationMode: "spec-driven" as OrchestrationMode }
          : {}),
      });
      toast("Run created", "success");
      navigate(`/run/${run.id}`);
    } catch {
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
            onBlur={handlePromptBlur}
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

        {/* Classification fields */}
        <div>
          <div className="mb-1 flex items-center gap-2">
            <label className="block text-xs text-muted-foreground">Classification</label>
            {classifying && (
              <span className="text-xs text-muted-foreground animate-pulse">Auto-suggesting...</span>
            )}
          </div>
          <div className="flex gap-2">
            <Input
              className="flex-1"
              value={project}
              onChange={(e) => {
                setProject(e.target.value);
                userEditedProject.current = true;
              }}
              placeholder="Project"
            />
            <Input
              className="flex-1"
              value={feature}
              onChange={(e) => {
                setFeature(e.target.value);
                userEditedFeature.current = true;
              }}
              placeholder="Feature"
            />
          </div>
          <Input
            className="mt-2"
            value={tags}
            onChange={(e) => {
              setTags(e.target.value);
              userEditedTags.current = true;
            }}
            placeholder="Tags (comma-separated)"
          />
        </div>

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
