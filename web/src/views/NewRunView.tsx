import { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useClient, mapRun } from "../hooks/useClient";
import { apiFetch } from "../hooks/apiFetch";
import { useToast } from "../components/Toast";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import MarkdownEditor from "../components/MarkdownEditor";
import {
  MODEL_TIER_OPTIONS,
  ORCHESTRATION_MODE_OPTIONS,
  type OrchestrationMode,
} from "../types/agent-run";

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

  // Configurable settings
  const [modelTier, setModelTier] = useState("default");
  const [ttlMinutes, setTtlMinutes] = useState(15);
  const [orchestrationMode, setOrchestrationMode] = useState<OrchestrationMode>("single");
  const [implementModelTier, setImplementModelTier] = useState("");

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
      if (run.spec.orchestrationMode) {
        setOrchestrationMode(run.spec.orchestrationMode);
      }
      if (run.spec.modelTier) {
        setModelTier(run.spec.modelTier);
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

  // When switching to spec mode, default to progressive orchestration
  useEffect(() => {
    if (mode === "spec") {
      setOrchestrationMode("spec-driven");
    }
  }, [mode]);

  // Auto-classify prompt (called when prompt changes significantly)
  const classifyPrompt = useCallback(async () => {
    if (prompt.trim().length <= 10) return;
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
      // Classification is best-effort
    } finally {
      setClassifying(false);
    }
  }, [prompt, repo, branch]);

  // Determine the effective prompt: for spec mode, spec content is the prompt if prompt is empty
  const effectivePrompt = mode === "spec" && !prompt.trim() && specContent.trim()
    ? specContent.trim()
    : prompt.trim();

  const canRun = effectivePrompt && repo.trim();

  async function handleRun() {
    if (!canRun) return;
    setSubmitting(true);
    // Best-effort classification before submit
    await classifyPrompt();
    try {
      const parsedTags = tags
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean);

      const run = await client.createAgentRun({
        backend: "pod",
        repos: [{ url: repo.trim(), branch: branch.trim() || "main" }],
        prompt: effectivePrompt,
        ttlSeconds: ttlMinutes * 60,
        modelTier,
        orchestrationMode,
        ...(project.trim() ? { project: project.trim() } : {}),
        ...(feature.trim() ? { feature: feature.trim() } : {}),
        ...(parsedTags.length > 0 ? { tags: parsedTags } : {}),
        ...(mode === "spec" && specContent.trim()
          ? { specContent: specContent.trim() }
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

  const selectedModel = MODEL_TIER_OPTIONS.find((m) => m.value === modelTier);
  const selectedMode = ORCHESTRATION_MODE_OPTIONS.find((m) => m.value === orchestrationMode);

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
        <div className={mode === "spec" ? "" : "flex-1"}>
          <label className="mb-1 block text-xs text-muted-foreground">
            {mode === "prompt" ? "Prompt" : "Prompt (optional — spec content is used if empty)"}
          </label>
          <MarkdownEditor
            value={prompt}
            onChange={(v) => { setPrompt(v); }}
            placeholder={mode === "spec" ? "Optional override prompt" : "What should the agent do?"}
            minHeight={mode === "spec" ? "80px" : "200px"}
            autoFocus
          />
        </div>

        {/* Spec editor (visible in spec mode) */}
        {mode === "spec" && (
          <div className="flex-1">
            <label className="mb-1 block text-xs text-muted-foreground">Spec</label>
            <MarkdownEditor
              value={specContent}
              onChange={(v) => setSpecContent(v)}
              placeholder="Paste your spec (markdown)..."
              minHeight="300px"
            />
          </div>
        )}

        {/* Configuration row: Model, Timeout, Orchestration */}
        <div>
          <label className="mb-1 block text-xs text-muted-foreground">Configuration</label>
          <div className="flex gap-2">
            {/* Model selector */}
            <div className="flex-1">
              <select
                className="w-full border bg-background px-2 py-1.5 text-sm outline-none focus:border-primary"
                value={modelTier}
                onChange={(e) => setModelTier(e.target.value)}
              >
                {MODEL_TIER_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Timeout */}
            <div className="w-24">
              <div className="flex items-center border bg-background">
                <input
                  type="number"
                  min={1}
                  max={120}
                  className="w-full bg-transparent px-2 py-1.5 text-sm outline-none"
                  value={ttlMinutes}
                  onChange={(e) => setTtlMinutes(Math.max(1, Math.min(120, Number(e.target.value) || 15)))}
                />
                <span className="pr-2 text-xs text-muted-foreground">min</span>
              </div>
            </div>

            {/* Orchestration mode */}
            <div className="flex-1">
              <select
                className="w-full border bg-background px-2 py-1.5 text-sm outline-none focus:border-primary"
                value={orchestrationMode}
                onChange={(e) => setOrchestrationMode(e.target.value as OrchestrationMode)}
              >
                {ORCHESTRATION_MODE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Config descriptions */}
          <div className="mt-1 flex gap-2 text-[10px] text-muted-foreground">
            <span className="flex-1 truncate">
              {orchestrationMode === "spec-driven" ? "Manage: " : ""}{selectedModel?.description}
            </span>
            <span className="w-24" />
            <span className="flex-1 truncate">{selectedMode?.description}</span>
          </div>

          {/* Implement model (Progressive mode only) */}
          {orchestrationMode === "spec-driven" && (
            <div className="mt-2">
              <div className="flex gap-2 items-center">
                <label className="text-[10px] text-muted-foreground w-24 shrink-0">Implement model</label>
                <select
                  className="flex-1 border bg-background px-2 py-1.5 text-sm outline-none focus:border-primary"
                  value={implementModelTier}
                  onChange={(e) => setImplementModelTier(e.target.value)}
                >
                  <option value="">Same as manage</option>
                  {MODEL_TIER_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>
              </div>
              {implementModelTier && (
                <div className="mt-0.5 text-[10px] text-muted-foreground ml-24 pl-2">
                  {MODEL_TIER_OPTIONS.find((m) => m.value === implementModelTier)?.description}
                </div>
              )}
            </div>
          )}
        </div>

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

        {/* Actions */}
        <div className="flex items-center justify-end gap-2">
          <Button variant="ghost" onClick={() => navigate("/")}>
            Cancel
          </Button>
          <Button onClick={handleRun} disabled={submitting || !canRun}>
            {submitting ? "Creating..." : "Run"}
          </Button>
        </div>
      </div>
    </div>
  );
}
