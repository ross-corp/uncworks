import { useState, useEffect } from "react";
import { useNavigate, Link } from "react-router-dom";
import { toast } from "sonner";
import { apiFetch } from "../hooks/apiFetch";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";

interface Template {
  metadata: { name: string };
  spec: { displayName?: string };
}

interface Project {
  name: string;
  displayName?: string;
}

interface Step {
  name: string;
  templateRef: string;
  dependsOn: string[];
}

export default function ChainNewView() {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [projectRef, setProjectRef] = useState("");
  const [steps, setSteps] = useState<Step[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    apiFetch("/api/v1/templates")
      .then((r) => r.ok ? r.json() : [])
      .then(setTemplates)
      .catch(() => {});
    apiFetch("/api/v1/projects")
      .then((r) => r.ok ? r.json() : [])
      .then(setProjects)
      .catch(() => {});
  }, []);

  function addStep() {
    setSteps((prev) => [...prev, { name: "", templateRef: "", dependsOn: [] }]);
  }

  function removeStep(idx: number) {
    const removedName = steps[idx].name;
    setSteps((prev) =>
      prev
        .filter((_, i) => i !== idx)
        .map((s) => ({ ...s, dependsOn: s.dependsOn.filter((d) => d !== removedName) }))
    );
  }

  function updateStep(idx: number, field: keyof Step, value: string | string[]) {
    setSteps((prev) => prev.map((s, i) => i === idx ? { ...s, [field]: value } : s));
  }

  function toggleDependsOn(stepIdx: number, depName: string) {
    const current = steps[stepIdx].dependsOn;
    const next = current.includes(depName)
      ? current.filter((d) => d !== depName)
      : [...current, depName];
    updateStep(stepIdx, "dependsOn", next);
  }

  const isValid =
    name.trim() !== "" &&
    steps.length > 0 &&
    steps.every((s) => s.name.trim() !== "" && s.templateRef !== "");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!isValid) return;
    setSubmitting(true);
    try {
      const resp = await apiFetch("/api/v1/chains", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim(),
          spec: {
            displayName: displayName.trim() || undefined,
            description: description.trim() || undefined,
            projectRef: projectRef || undefined,
            steps: steps.map((s) => ({
              name: s.name.trim(),
              templateRef: s.templateRef,
              dependsOn: s.dependsOn.length > 0 ? s.dependsOn : undefined,
            })),
          },
        }),
      });
      if (resp.ok) {
        toast.success("Chain created");
        navigate("/chains");
      } else {
        const data = await resp.json().catch(() => ({}));
        toast.error(data.error || "Failed to create chain");
      }
    } catch {
      toast.error("Failed to create chain");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <span className="text-xs text-muted-foreground">
          Templates /{" "}
          <Link to="/chains" className="hover:text-foreground transition-colors">
            Chains
          </Link>
          {" / New Chain"}
        </span>
        <Badge variant="secondary" className="ml-auto text-xs">new</Badge>
      </div>

      <div className="flex-1 overflow-y-auto">
        <form onSubmit={handleSubmit} className="max-w-2xl p-4 space-y-4">
          {/* Base fields */}
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Name <span className="text-destructive">*</span>
            </label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-chain"
              required
            />
            <p className="text-xs text-muted-foreground">Slug identifier (required)</p>
          </div>

          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Display Name
            </label>
            <Input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="My Chain"
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Description
            </label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional description"
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Project
            </label>
            <select
              value={projectRef}
              onChange={(e) => setProjectRef(e.target.value)}
              className="w-full border border-input bg-background px-3 py-2 text-sm rounded-none focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="">— none —</option>
              {projects.map((p) => (
                <option key={p.name} value={p.name}>
                  {p.displayName || p.name}
                </option>
              ))}
            </select>
          </div>

          {/* Steps builder */}
          <div className="space-y-2 pt-2">
            <div className="flex items-center justify-between">
              <span className="text-sm font-semibold">Steps</span>
              <Button type="button" size="sm" onClick={addStep}>
                + add step
              </Button>
            </div>

            {steps.length === 0 && (
              <p className="text-xs text-muted-foreground">No steps yet. Add at least one step.</p>
            )}

            {steps.map((step, idx) => {
              const prevNames = steps.slice(0, idx).map((s) => s.name).filter((n) => n.trim() !== "");
              return (
                <div key={idx} className="border border-border/60 p-3 space-y-2">
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground w-5 shrink-0">{idx + 1}.</span>
                    <Input
                      value={step.name}
                      onChange={(e) => updateStep(idx, "name", e.target.value)}
                      placeholder="step-name"
                      className="flex-1"
                    />
                    <select
                      value={step.templateRef}
                      onChange={(e) => updateStep(idx, "templateRef", e.target.value)}
                      className="flex-1 border border-input bg-background px-3 py-2 text-sm rounded-none focus:outline-none focus:ring-1 focus:ring-primary"
                    >
                      <option value="">— template —</option>
                      {templates.map((t) => (
                        <option key={t.metadata.name} value={t.metadata.name}>
                          {t.spec.displayName || t.metadata.name}
                        </option>
                      ))}
                    </select>
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      onClick={() => removeStep(idx)}
                    >
                      ×
                    </Button>
                  </div>
                  {prevNames.length > 0 && (
                    <div className="pl-7 space-y-1">
                      <span className="text-xs text-muted-foreground">Depends on:</span>
                      <div className="flex flex-wrap gap-3">
                        {prevNames.map((pn) => (
                          <label key={pn} className="flex items-center gap-1 text-xs cursor-pointer">
                            <input
                              type="checkbox"
                              checked={step.dependsOn.includes(pn)}
                              onChange={() => toggleDependsOn(idx, pn)}
                            />
                            {pn}
                          </label>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>

          <div className="pt-2">
            <Button type="submit" disabled={!isValid || submitting} size="sm">
              {submitting ? "Creating…" : "Create Chain"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
