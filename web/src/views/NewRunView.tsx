import { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useClient, mapRun } from "../hooks/useClient";
import { apiFetch } from "../hooks/apiFetch";
import { useToast } from "../components/Toast";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
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
  const [repos, setRepos] = useState([{ url: "https://github.com/roshbhatia/neph.nvim", branch: "main" }]);
  const [submitting, setSubmitting] = useState(false);
  const [mode, setMode] = useState<"prompt" | "spec">("prompt");
  const [specContent, setSpecContent] = useState("");

  // Configuration
  const [modelTier, setModelTier] = useState("default");
  const [ttlMinutes, setTtlMinutes] = useState(15);
  const [orchestrationMode, setOrchestrationMode] = useState<OrchestrationMode>("single");
  const [implementModelTier, setImplementModelTier] = useState("");

  // AI improvement
  const [improvingPrompt, setImprovingPrompt] = useState(false);
  const [improvingSpec, setImprovingSpec] = useState(false);

  // Classification
  const [project, setProject] = useState("");
  const [feature, setFeature] = useState("");
  const [tags, setTags] = useState("");
  const [classifying, setClassifying] = useState(false);

  // Existing projects/features for dropdown suggestions
  const [existingProjects, setExistingProjects] = useState<string[]>([]);
  const [existingFeatures, setExistingFeatures] = useState<string[]>([]);

  const userEditedProject = useRef(false);
  const userEditedFeature = useRef(false);
  const userEditedTags = useRef(false);

  // Fetch existing projects/features for suggestions
  useEffect(() => {
    client.listAgentRuns().then((runs) => {
      const projects = new Set<string>();
      const features = new Set<string>();
      for (const r of runs) {
        if (r.spec.project) projects.add(r.spec.project as string);
        if (r.spec.feature) features.add(r.spec.feature as string);
      }
      setExistingProjects(Array.from(projects).sort());
      setExistingFeatures(Array.from(features).sort());
    }).catch(() => {});
  }, [client]);

  // Clone support
  useEffect(() => {
    const cloneId = searchParams.get("clone");
    if (!cloneId) return;
    client.getAgentRun(cloneId).then((raw) => {
      const run = mapRun(raw);
      setPrompt(run.spec.prompt || "");
      if (run.spec.repos?.length) {
        setRepos(run.spec.repos.map((r) => ({ url: r.url, branch: r.branch })));
      }
      if (run.spec.specContent) {
        setSpecContent(run.spec.specContent);
        setMode("spec");
      }
      if (run.spec.orchestrationMode) setOrchestrationMode(run.spec.orchestrationMode);
      if (run.spec.modelTier) setModelTier(run.spec.modelTier);
      if (run.spec.project) { setProject(run.spec.project); userEditedProject.current = true; }
      if (run.spec.feature) { setFeature(run.spec.feature); userEditedFeature.current = true; }
      if (run.spec.tags?.length) { setTags(run.spec.tags.join(", ")); userEditedTags.current = true; }
    }).catch(() => {});
  }, [searchParams, client]);

  // Pre-fill from project query param
  useEffect(() => {
    const projName = searchParams.get("project");
    if (projName) {
      setProject(projName);
      userEditedProject.current = true;
      // Fetch project details for defaults
      apiFetch(`/api/v1/projects/${projName}`).then(async (resp) => {
        if (!resp.ok) return;
        const proj = await resp.json();
        if (proj.repos?.length) setRepos(proj.repos.map((r: { url: string; branch: string }) => ({ url: r.url, branch: r.branch || "main" })));
        if (proj.defaults?.modelTier) setModelTier(proj.defaults.modelTier);
        if (proj.defaults?.orchestrationMode) setOrchestrationMode(proj.defaults.orchestrationMode);
      }).catch(() => {});
    }
    const specName = searchParams.get("spec");
    if (specName) {
      setMode("spec");
      setOrchestrationMode("spec-driven");
    }
  }, [searchParams]);

  useEffect(() => {
    if (mode === "spec") setOrchestrationMode("spec-driven");
  }, [mode]);

  const classifyPrompt = useCallback(async () => {
    if (prompt.trim().length <= 10) return;
    if (userEditedProject.current && userEditedFeature.current && userEditedTags.current) return;
    setClassifying(true);
    try {
      const resp = await apiFetch("/api/v1/classify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ prompt: prompt.trim(), repos: repos.filter((r) => r.url.trim()) }),
      });
      if (!resp.ok) return;
      const data = await resp.json() as { project?: string; feature?: string; tags?: string[] };
      if (data.project && !userEditedProject.current) setProject(data.project);
      if (data.feature && !userEditedFeature.current) setFeature(data.feature);
      if (data.tags?.length && !userEditedTags.current) setTags(data.tags.join(", "));
    } catch { /* silent */ } finally { setClassifying(false); }
  }, [prompt, repos]);

  async function improveText(text: string, kind: "prompt" | "spec", setter: (v: string) => void, setLoading: (v: boolean) => void) {
    if (!text.trim()) return;
    setLoading(true);
    try {
      const resp = await apiFetch("/api/v1/improve-text", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text, kind }),
      });
      if (resp.ok) {
        const data = await resp.json();
        if (data.improved) setter(data.improved);
      }
    } catch { /* silent */ }
    setLoading(false);
  }

  const effectivePrompt = mode === "spec" && !prompt.trim() && specContent.trim()
    ? specContent.trim()
    : prompt.trim();

  const canRun = effectivePrompt && repos.some((r) => r.url.trim());

  async function handleRun() {
    if (!canRun) return;
    setSubmitting(true);
    await classifyPrompt();
    try {
      const parsedTags = tags.split(",").map((t) => t.trim()).filter(Boolean);
      const validRepos = repos.filter((r) => r.url.trim()).map((r) => ({ url: r.url.trim(), branch: r.branch.trim() || "main" }));

      const run = await client.createAgentRun({
        backend: "pod",
        repos: validRepos,
        prompt: effectivePrompt,
        ttlSeconds: ttlMinutes * 60,
        modelTier,
        orchestrationMode,
        ...(project.trim() ? { project: project.trim() } : {}),
        ...(feature.trim() ? { feature: feature.trim() } : {}),
        ...(parsedTags.length > 0 ? { tags: parsedTags } : {}),
        ...(mode === "spec" && specContent.trim() ? { specContent: specContent.trim() } : {}),
      });
      toast("Run created", "success");
      navigate(`/run/${run.id}`);
    } catch { toast("Failed to create run", "error"); }
    finally { setSubmitting(false); }
  }

  function addRepo() { setRepos([...repos, { url: "", branch: "main" }]); }
  function removeRepo(i: number) { setRepos(repos.filter((_, idx) => idx !== i)); }
  function updateRepo(i: number, field: "url" | "branch", value: string) {
    setRepos(repos.map((r, idx) => idx === i ? { ...r, [field]: value } : r));
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-4 py-2">
        <span className="font-semibold">New Run</span>
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">ctrl+enter to run</span>
          <Button size="sm" variant="ghost" className="h-6 text-[11px]" onClick={() => navigate("/")}>
            Cancel
          </Button>
        </div>
      </div>

      {/* Form — two-column layout */}
      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-3xl p-4 space-y-4">

          {/* Repositories */}
          <section>
            <label className="text-xs text-muted-foreground mb-1 block">Repositories</label>
            {repos.map((r, i) => (
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
                {repos.length > 1 && (
                  <Button size="sm" variant="ghost" className="h-8 px-2 text-muted-foreground" onClick={() => removeRepo(i)}>
                    x
                  </Button>
                )}
              </div>
            ))}
            <Button size="sm" variant="ghost" className="h-6 text-[11px] text-muted-foreground" onClick={addRepo}>
              + add repository
            </Button>
          </section>

          {/* Mode + Prompt */}
          <section>
            <div className="flex items-center gap-2 mb-1">
              <label className="text-xs text-muted-foreground">
                {mode === "prompt" ? "Prompt" : "Prompt (optional when spec is provided)"}
              </label>
              <div className="flex gap-0.5 ml-auto">
                {(["prompt", "spec"] as const).map((m) => (
                  <button
                    key={m}
                    onClick={() => setMode(m)}
                    className={`px-2 py-0.5 text-[10px] rounded transition-colors ${
                      mode === m ? "bg-foreground text-background" : "text-muted-foreground hover:text-foreground"
                    }`}
                  >
                    {m}
                  </button>
                ))}
              </div>
            </div>
            <MarkdownEditor
              value={prompt}
              onChange={setPrompt}
              placeholder="What should the agent do?"
              minHeight={mode === "spec" ? "60px" : "120px"}
              autoFocus
            />
            {prompt.trim().length > 10 && (
              <div className="flex justify-end mt-1">
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-6 text-[11px]"
                  disabled={improvingPrompt}
                  onClick={() => improveText(prompt, "prompt", setPrompt, setImprovingPrompt)}
                >
                  {improvingPrompt ? "Improving..." : "Improve with AI"}
                </Button>
              </div>
            )}
          </section>

          {/* Spec editor */}
          {mode === "spec" && (
            <section>
              <label className="text-xs text-muted-foreground mb-1 block">Spec (markdown)</label>
              <MarkdownEditor
                value={specContent}
                onChange={setSpecContent}
                placeholder="Paste or write your spec..."
                minHeight="180px"
              />
              {specContent.trim().length > 10 && (
                <div className="flex justify-end mt-1">
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-6 text-[11px]"
                    disabled={improvingSpec}
                    onClick={() => improveText(specContent, "spec", setSpecContent, setImprovingSpec)}
                  >
                    {improvingSpec ? "Improving..." : "Improve with AI"}
                  </Button>
                </div>
              )}
            </section>
          )}

          {/* Configuration — horizontal row */}
          <section>
            <label className="text-xs text-muted-foreground mb-1 block">Configuration</label>
            <div className="flex gap-2 items-start">
              <div className="flex-1">
                <Select value={modelTier} onValueChange={setModelTier}>
                  <SelectTrigger size="sm" className="w-full h-8">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {MODEL_TIER_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        <span>{opt.label}</span>
                        <span className="text-muted-foreground ml-1 text-[10px]">{opt.description}</span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Input
                type="number"
                min={1} max={120}
                className="w-16 h-8 text-sm text-center"
                value={ttlMinutes}
                onChange={(e) => setTtlMinutes(Math.max(1, Math.min(120, Number(e.target.value) || 15)))}
                title="Timeout (minutes)"
              />
              <div className="flex-1">
                <Select value={orchestrationMode} onValueChange={(v) => setOrchestrationMode(v as OrchestrationMode)}>
                  <SelectTrigger size="sm" className="w-full h-8">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {ORCHESTRATION_MODE_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        <span>{opt.label}</span>
                        <span className="text-muted-foreground ml-1 text-[10px]">{opt.description}</span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* Implement model (progressive only) */}
            {orchestrationMode === "spec-driven" && (
              <div className="flex gap-2 items-center mt-2">
                <span className="text-[10px] text-muted-foreground shrink-0">Implement model</span>
                <Select value={implementModelTier || "__same__"} onValueChange={(v) => setImplementModelTier(v === "__same__" ? "" : v)}>
                  <SelectTrigger size="sm" className="h-8 flex-1">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__same__">Same as above</SelectItem>
                    {MODEL_TIER_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>{opt.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
          </section>

          {/* Classification — project, feature, tags */}
          <section>
            <div className="flex items-center gap-2 mb-1">
              <label className="text-xs text-muted-foreground">Classification</label>
              {classifying && <span className="text-[10px] text-muted-foreground animate-pulse">suggesting...</span>}
            </div>
            <div className="flex gap-2">
              {/* Project dropdown with suggestions */}
              <div className="flex-1">
                <Input
                  className="h-8 text-sm"
                  value={project}
                  onChange={(e) => { setProject(e.target.value); userEditedProject.current = true; }}
                  placeholder="Project"
                  list="project-suggestions"
                />
                <datalist id="project-suggestions">
                  {existingProjects.map((p) => <option key={p} value={p} />)}
                </datalist>
              </div>
              {/* Feature dropdown with suggestions */}
              <div className="flex-1">
                <Input
                  className="h-8 text-sm"
                  value={feature}
                  onChange={(e) => { setFeature(e.target.value); userEditedFeature.current = true; }}
                  placeholder="Feature"
                  list="feature-suggestions"
                />
                <datalist id="feature-suggestions">
                  {existingFeatures.map((f) => <option key={f} value={f} />)}
                </datalist>
              </div>
            </div>
            <Input
              className="mt-1 h-8 text-sm"
              value={tags}
              onChange={(e) => { setTags(e.target.value); userEditedTags.current = true; }}
              placeholder="Tags (comma-separated)"
            />
          </section>

          {/* Submit */}
          <div className="flex items-center justify-end gap-2 pt-2 pb-4">
            <Button variant="ghost" onClick={() => navigate("/")}>
              Cancel
            </Button>
            <Button onClick={handleRun} disabled={submitting || !canRun}>
              {submitting ? "Creating..." : "Run"}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
