import { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate, useSearchParams, Link } from "react-router-dom";
import { useClient, mapRun } from "../hooks/useClient";
import { apiFetch } from "../hooks/apiFetch";
import { useRunForm } from "../hooks/useRunForm";
import { useSettings } from "../hooks/useSettings";
import { toast } from "sonner";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../components/ui/tabs";
import MarkdownEditor from "../components/MarkdownEditor";
import {
  MODEL_TIER_OPTIONS,
  ORCHESTRATION_MODE_OPTIONS,
  type OrchestrationMode,
} from "../types/agent-run";

interface ProjectOption {
  name: string;
  displayName: string;
  repos: { url: string; branch: string }[];
  defaults?: {
    modelTier?: string;
    orchestrationMode?: string;
  };
}

export default function NewRunView() {
  const client = useClient();
  const navigate = useNavigate();
  const { configStatus, loading: settingsLoading } = useSettings();

  const [searchParams] = useSearchParams();
  const { form, set, reset } = useRunForm();

  // Destructure form fields for readability
  const {
    prompt, repos, mode, specContent,
    modelTier, ttlMinutes, orchestrationMode, implementModelTier,
    projectRef, specRef, customLabelMode,
    project, feature, tags,
  } = form;

  // Auxiliary UI state (not part of the run creation payload)
  const [submitting, setSubmitting] = useState(false);
  const [improvingPrompt, setImprovingPrompt] = useState(false);
  const [improvingSpec, setImprovingSpec] = useState(false);
  const [availableProjects, setAvailableProjects] = useState<ProjectOption[]>([]);
  const [classifying, setClassifying] = useState(false);
  const [existingProjects, setExistingProjects] = useState<string[]>([]);
  const [existingFeatures, setExistingFeatures] = useState<string[]>([]);

  const userEditedProject = useRef(false);
  const userEditedFeature = useRef(false);
  const userEditedTags = useRef(false);

  // Fetch Project CRDs for the project selector
  useEffect(() => {
    const ac = new AbortController();
    apiFetch("/api/v1/projects", { signal: ac.signal }).then(async (resp) => {
      if (resp.ok) {
        const data = await resp.json() as ProjectOption[];
        setAvailableProjects(data);
      }
    }).catch((err) => {
      if (err instanceof Error && err.name === "AbortError") return;
      console.error("[NewRunView]", err); toast.error("Failed to load projects");
    });
    return () => ac.abort();
  }, []);

  // Fetch existing projects/features for suggestions
  useEffect(() => {
    let cancelled = false;
    client.listAgentRuns().then((runs) => {
      if (cancelled) return;
      const projects = new Set<string>();
      const features = new Set<string>();
      for (const r of runs) {
        if (r.spec.project) projects.add(r.spec.project as string);
        if (r.spec.feature) features.add(r.spec.feature as string);
      }
      setExistingProjects(Array.from(projects).sort());
      setExistingFeatures(Array.from(features).sort());
    }).catch((err) => {
      if (cancelled) return;
      console.error("[NewRunView]", err); toast.error("Failed to load run history");
    });
    return () => { cancelled = true; };
  }, [client]);

  // Clone support
  useEffect(() => {
    const cloneId = searchParams.get("clone");
    if (!cloneId) return;
    let cancelled = false;
    client.getAgentRun(cloneId).then((raw) => {
      if (cancelled) return;
      const run = mapRun(raw);
      set.prompt(run.spec.prompt || "");
      if (run.spec.repos?.length) {
        set.repos(run.spec.repos.map((r) => ({ url: r.url, branch: r.branch })));
      }
      if (run.spec.specContent) {
        set.specContent(run.spec.specContent);
        set.mode("spec");
      }
      if (run.spec.orchestrationMode) set.orchestrationMode(run.spec.orchestrationMode);
      if (run.spec.modelTier) set.modelTier(run.spec.modelTier);
      if (run.spec.project) { set.project(run.spec.project); userEditedProject.current = true; }
      if (run.spec.feature) { set.feature(run.spec.feature); userEditedFeature.current = true; }
      if (run.spec.tags?.length) { set.tags(run.spec.tags.join(", ")); userEditedTags.current = true; }
    }).catch((err) => {
      if (cancelled) return;
      console.error("[NewRunView]", err); toast.error("Failed to load run to clone");
    });
    return () => { cancelled = true; };
  }, [searchParams, client, set]);

  // Pre-fill from project/spec query params
  useEffect(() => {
    const projName = searchParams.get("project");
    const specName = searchParams.get("spec");
    let cancelled = false;
    const ac = new AbortController();

    if (projName) {
      set.projectRef(projName);
      set.project(projName);
      userEditedProject.current = true;
      // Fetch project details for defaults
      apiFetch(`/api/v1/projects/${projName}`, { signal: ac.signal }).then(async (resp) => {
        if (!resp.ok || cancelled) return;
        const proj = await resp.json();
        if (cancelled) return;
        if (proj.repos?.length) set.repos(proj.repos.map((r: { url: string; branch: string }) => ({ url: r.url, branch: r.branch || "main" })));
        if (proj.defaults?.modelTier) set.modelTier(proj.defaults.modelTier);
        if (proj.defaults?.orchestrationMode) set.orchestrationMode(proj.defaults.orchestrationMode);
        if (proj.displayName || proj.name) set.project(proj.displayName || proj.name);
      }).catch((err) => {
        if (err instanceof Error && err.name === "AbortError") return;
        console.error("[NewRunView]", err);
      });
    }
    if (specName) {
      set.specRef(specName);
      set.mode("spec");
      set.orchestrationMode("spec-driven");
    }
    return () => { cancelled = true; ac.abort(); };
  }, [searchParams, set]);

  useEffect(() => {
    if (mode === "spec") set.orchestrationMode("spec-driven");
  }, [mode, set]);

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
      if (data.project && !userEditedProject.current) set.project(data.project);
      if (data.feature && !userEditedFeature.current) set.feature(data.feature);
      if (data.tags?.length && !userEditedTags.current) set.tags(data.tags.join(", "));
    } catch (err) { console.error("[NewRunView]", err); toast.error("Classification failed"); } finally { setClassifying(false); }
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
    } catch (err) { console.error("[NewRunView]", err); toast.error("Couldn't improve prompt — try again"); }
    setLoading(false);
  }

  const effectivePrompt = mode === "spec" && !prompt.trim() && specContent.trim()
    ? specContent.trim()
    : prompt.trim();

  const canRun = effectivePrompt && repos.some((r) => r.url.trim());

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter" && canRun && !submitting) {
        e.preventDefault();
        handleRun();
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canRun, submitting]);

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
        ...(projectRef.trim() ? { projectRef: projectRef.trim() } : {}),
        ...(specRef.trim() ? { specRef: specRef.trim() } : {}),
      });
      toast.success("Run created");
      reset();
      navigate(`/run/${run.id}`);
    } catch (err) { console.error("[NewRunView]", err); toast.error("Failed to create run"); }
    finally { setSubmitting(false); }
  }

  function handleProjectRefChange(name: string) {
    set.projectRef(name);
    set.specRef(""); // clear stale spec ref when project changes
    if (!name) return;
    const proj = availableProjects.find((p) => p.name === name);
    if (!proj) return;
    if (proj.repos?.length) set.repos(proj.repos.map((r) => ({ url: r.url, branch: r.branch || "main" })));
    if (proj.defaults?.modelTier) set.modelTier(proj.defaults.modelTier);
    if (proj.defaults?.orchestrationMode) set.orchestrationMode(proj.defaults.orchestrationMode as OrchestrationMode);
    if (!userEditedProject.current) {
      set.project(proj.displayName || proj.name);
    }
  }

  function addRepo() { set.repos([...repos, { url: "", branch: "main" }]); }
  function removeRepo(i: number) { set.repos(repos.filter((_, idx) => idx !== i)); }
  function updateRepo(i: number, field: "url" | "branch", value: string) {
    set.repos(repos.map((r, idx) => idx === i ? { ...r, [field]: value } : r));
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <div className="flex items-center gap-3 flex-1">
          <span className="font-semibold text-base">New Run</span>
          {/* Prompt / Spec mode toggle — segmented control in header */}
          <div className="flex gap-0.5 bg-muted/50 rounded-md p-0.5">
            {(["prompt", "spec"] as const).map((m) => (
              <button
                key={m}
                onClick={() => set.mode(m)}
                className={`px-2.5 py-1 text-xs rounded-md transition-colors capitalize ${
                  mode === m ? "bg-background text-foreground shadow-sm font-medium" : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {m}
              </button>
            ))}
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Button size="sm" variant="ghost" onClick={() => navigate("/")}>
            Cancel
          </Button>
        </div>
      </div>

      {/* Config gate: warn when LLM key is missing */}
      {!settingsLoading && !configStatus.hasLLMKey && (
        <div className="border-b bg-amber-500/5 border-amber-500/30 px-4 py-2.5 flex items-center gap-3">
          <span className="text-amber-500 shrink-0 text-sm">&#9888;</span>
          <span className="text-xs text-muted-foreground flex-1">
            No LLM API key configured. Runs will fail without one.
          </span>
          <Link
            to="/settings"
            className="text-xs text-amber-600 dark:text-amber-400 hover:underline shrink-0"
          >
            Configure &rarr;
          </Link>
        </div>
      )}

      {/* Tabs */}
      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-2xl p-6 space-y-6">
          <Tabs defaultValue="core">
            <TabsList>
              <TabsTrigger value="core">Core</TabsTrigger>
              <TabsTrigger value="config">Config</TabsTrigger>
            </TabsList>

            {/* Core tab: prompt/spec editor + repositories */}
            <TabsContent value="core">
              <div className="space-y-6 pt-4">

                {/* Unified project field */}
                <section>
                  <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wider">Project</label>
                  {customLabelMode ? (
                    <div className="flex gap-2">
                      <Input
                        className="flex-1 h-8 text-sm"
                        value={project}
                        onChange={(e) => { set.project(e.target.value); userEditedProject.current = true; }}
                        placeholder="Custom project label"
                        autoFocus
                        list="project-suggestions"
                      />
                      <datalist id="project-suggestions">
                        {existingProjects.map((p) => <option key={p} value={p} />)}
                      </datalist>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="px-2 text-muted-foreground"
                        onClick={() => { set.customLabelMode(false); set.project(""); userEditedProject.current = false; }}
                      >
                        ✕
                      </Button>
                    </div>
                  ) : (
                    <Select
                      value={projectRef || "__none__"}
                      onValueChange={(v) => {
                        if (v === "__custom__") {
                          set.projectRef("");
                          set.customLabelMode(true);
                          return;
                        }
                        set.customLabelMode(false);
                        handleProjectRefChange(v === "__none__" ? "" : v);
                      }}
                    >
                      <SelectTrigger size="sm" className="w-full h-8">
                        <SelectValue placeholder="None (standalone run)" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="__none__">None (standalone run)</SelectItem>
                        {availableProjects.map((p) => (
                          <SelectItem key={p.name} value={p.name}>
                            {p.displayName || p.name}
                          </SelectItem>
                        ))}
                        <SelectItem value="__custom__">Custom label...</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                  {projectRef && !customLabelMode && (
                    <p className="text-xs text-muted-foreground mt-1">
                      Repos, model, and defaults inherited from project (overridable below).
                      {specRef && <span className="ml-1">Spec: <span className="font-mono">{specRef}</span></span>}
                    </p>
                  )}
                </section>

                {/* Prompt */}
                <section>
                  <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wider">
                    {mode === "prompt" ? "Prompt" : "Prompt (optional with spec)"}
                  </label>
                  <MarkdownEditor
                    value={prompt}
                    onChange={set.prompt}
                    placeholder="What should the agent do?"
                    minHeight={mode === "spec" ? "60px" : "120px"}
                    autoFocus
                  />
                  {prompt.trim().length > 10 && (
                    <div className="flex justify-end mt-1">
                      <Button
                        size="sm"
                        className="text-sm"
                        disabled={improvingPrompt}
                        onClick={() => improveText(prompt, "prompt", set.prompt, setImprovingPrompt)}
                      >
                        {improvingPrompt ? "Improving..." : "✨ Improve with AI"}
                      </Button>
                    </div>
                  )}
                </section>

                {/* Spec editor */}
                {mode === "spec" && (
                  <section>
                    <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wider">Spec (markdown)</label>
                    <MarkdownEditor
                      value={specContent}
                      onChange={set.specContent}
                      placeholder="Paste or write your spec..."
                      minHeight="180px"
                    />
                    {specContent.trim().length > 10 && (
                      <div className="flex justify-end mt-1">
                        <Button
                          size="sm"
                          className="text-sm"
                          disabled={improvingSpec}
                          onClick={() => improveText(specContent, "spec", set.specContent, setImprovingSpec)}
                        >
                          {improvingSpec ? "Improving..." : "✨ Improve with AI"}
                        </Button>
                      </div>
                    )}
                  </section>
                )}

                {/* Repositories */}
                <section>
                  <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wider">Repositories</label>
                  {repos.map((r, i) => (
                    <div key={i} className="flex gap-2 mb-1">
                      <Input
                        className="flex-1 h-8 text-sm"
                        value={r.url}
                        onChange={(e) => updateRepo(i, "url", e.target.value)}
                        placeholder="https://github.com/org/repo"
                      />
                      <Input
                        className="min-w-0 flex-1 h-8 text-sm"
                        value={r.branch}
                        onChange={(e) => updateRepo(i, "branch", e.target.value)}
                        placeholder="main"
                      />
                      {repos.length > 1 && (
                        <Button size="sm" variant="ghost" className="px-2 text-muted-foreground" onClick={() => removeRepo(i)}>
                          x
                        </Button>
                      )}
                    </div>
                  ))}
                  <Button size="sm" variant="ghost" className="text-xs text-muted-foreground" onClick={addRepo}>
                    + add repository
                  </Button>
                </section>

              </div>
            </TabsContent>

            {/* Config tab: model, TTL, orchestration, implement model, classification */}
            <TabsContent value="config">
              <div className="space-y-6 pt-4">

                {/* Configuration — horizontal row */}
                <section>
                  <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wider">Configuration</label>
                  <div className="flex gap-2 items-start">
                    <div className="flex-1">
                      <Select value={modelTier} onValueChange={set.modelTier}>
                        <SelectTrigger size="sm" className="w-full h-8">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {MODEL_TIER_OPTIONS.map((opt) => (
                            <SelectItem key={opt.value} value={opt.value}>
                              <span>{opt.label}</span>
                              <span className="text-muted-foreground ml-1 text-xs">{opt.description}</span>
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
                      onChange={(e) => set.ttlMinutes(Math.max(1, Math.min(120, Number(e.target.value) || 15)))}
                      title="Timeout (minutes)"
                    />
                    <div className="flex-1">
                      <Select value={orchestrationMode} onValueChange={(v) => set.orchestrationMode(v as OrchestrationMode)}>
                        <SelectTrigger size="sm" className="w-full h-8">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {ORCHESTRATION_MODE_OPTIONS.map((opt) => (
                            <SelectItem key={opt.value} value={opt.value}>
                              <span>{opt.label}</span>
                              <span className="text-muted-foreground ml-1 text-xs">{opt.description}</span>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  {/* Implement model (progressive only) */}
                  {orchestrationMode === "spec-driven" && (
                    <div className="flex gap-2 items-center mt-2">
                      <span className="text-xs text-muted-foreground shrink-0">Implement model</span>
                      <Select value={implementModelTier || "__same__"} onValueChange={(v) => set.implementModelTier(v === "__same__" ? "" : v)}>
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

                {/* Classification — feature, tags */}
                <section>
                  <div className="flex items-center gap-2 mb-1">
                    <label className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Classification</label>
                    {classifying && <span className="text-xs text-muted-foreground animate-pulse">suggesting...</span>}
                  </div>
                  <div className="flex gap-2">
                    {/* Feature dropdown with suggestions */}
                    <div className="flex-1">
                      <Input
                        className="text-sm"
                        value={feature}
                        onChange={(e) => { set.feature(e.target.value); userEditedFeature.current = true; }}
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
                    onChange={(e) => { set.tags(e.target.value); userEditedTags.current = true; }}
                    placeholder="Tags (comma-separated)"
                  />
                </section>

              </div>
            </TabsContent>
          </Tabs>

          {/* Submit — always visible, outside tabs */}
          <div className="flex items-center justify-end gap-2 pt-2 pb-4">
            <Button variant="ghost" onClick={() => navigate("/")}>
              Cancel
            </Button>
            <Button onClick={handleRun} disabled={submitting || !canRun}>
              {submitting ? "Creating..." : "Run"}
            </Button>
            {canRun && !submitting && (
              <span className="text-xs text-muted-foreground select-none">⌘↵</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
