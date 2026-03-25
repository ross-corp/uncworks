import { useState, useEffect, useMemo } from "react";
import { useNavigate, Link } from "react-router-dom";
import { toast } from "sonner";
import cronstrue from "cronstrue";
import { apiFetch } from "../hooks/apiFetch";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";

export default function ScheduleNewView() {
  const navigate = useNavigate();

  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [cron, setCron] = useState("");
  const [timezone, setTimezone] = useState("UTC");
  const [concurrencyPolicy, setConcurrencyPolicy] = useState("Forbid");

  const [targetType, setTargetType] = useState<"chain" | "template">("chain");
  const [chainRef, setChainRef] = useState("");
  const [templateRef, setTemplateRef] = useState("");
  const [chains, setChains] = useState<{ metadata: { name: string }; spec: { displayName?: string } }[]>([]);
  const [templates, setTemplates] = useState<{ metadata: { name: string }; spec: { displayName?: string } }[]>([]);

  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    apiFetch("/api/v1/chains").then((r) => { if (r.ok) r.json().then(setChains); }).catch(() => {});
    apiFetch("/api/v1/templates").then((r) => { if (r.ok) r.json().then(setTemplates); }).catch(() => {});
  }, []);

  const cronPreview = useMemo(() => {
    try {
      return cronstrue.toString(cron);
    } catch {
      return cron ? "Invalid cron expression" : "";
    }
  }, [cron]);

  const cronValid = useMemo(() => {
    if (!cron) return false;
    try {
      cronstrue.toString(cron);
      return true;
    } catch {
      return false;
    }
  }, [cron]);

  const submitDisabled =
    !name.trim() ||
    !cronValid ||
    (targetType === "chain" && !chainRef) ||
    (targetType === "template" && !templateRef) ||
    submitting;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (submitDisabled) return;
    setSubmitting(true);
    try {
      const spec: Record<string, string> = {
        cron,
        timezone,
        concurrencyPolicy,
      };
      if (displayName.trim()) spec.displayName = displayName.trim();
      if (targetType === "chain") {
        spec.chainRef = chainRef;
      } else {
        spec.templateRef = templateRef;
      }
      const resp = await apiFetch("/api/v1/schedules", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: name.trim(), spec }),
      });
      if (resp.ok) {
        toast.success("Schedule created");
        navigate("/schedules");
      } else {
        const data = await resp.json().catch(() => ({}));
        toast.error((data as { error?: string }).error || "Failed to create schedule");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create schedule");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <span className="text-sm text-muted-foreground">
          <Link to="/schedules" className="hover:text-foreground transition-colors">Schedules</Link>
        </span>
        <span className="text-muted-foreground">/</span>
        <span className="text-sm font-medium">New Schedule</span>
      </div>

      <div className="flex-1 overflow-y-auto">
        <form onSubmit={handleSubmit} className="max-w-2xl p-4 space-y-4">
          {/* name */}
          <div className="space-y-1">
            <label className="text-xs font-medium">Name <span className="text-destructive">*</span></label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-schedule"
            />
          </div>

          {/* displayName */}
          <div className="space-y-1">
            <label className="text-xs font-medium">Display Name</label>
            <Input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="My Schedule"
            />
          </div>

          {/* cron */}
          <div className="space-y-1">
            <label className="text-xs font-medium">Cron Expression <span className="text-destructive">*</span></label>
            <Input
              value={cron}
              onChange={(e) => setCron(e.target.value)}
              placeholder="0 * * * *"
              className="font-mono"
            />
            {cronPreview && (
              <p className={`text-xs ${cronValid ? "text-green-600 dark:text-green-400" : "text-destructive"}`}>
                {cronPreview}
              </p>
            )}
          </div>

          {/* timezone */}
          <div className="space-y-1">
            <label className="text-xs font-medium">Timezone</label>
            <Input
              value={timezone}
              onChange={(e) => setTimezone(e.target.value)}
              placeholder="UTC"
            />
          </div>

          {/* concurrencyPolicy */}
          <div className="space-y-1">
            <label className="text-xs font-medium">Concurrency Policy</label>
            <select
              value={concurrencyPolicy}
              onChange={(e) => setConcurrencyPolicy(e.target.value)}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            >
              <option value="Allow">Allow</option>
              <option value="Forbid">Forbid</option>
              <option value="Replace">Replace</option>
            </select>
          </div>

          {/* target type toggle */}
          <div className="space-y-2">
            <label className="text-xs font-medium">Target</label>
            <div className="flex gap-1 border rounded-md p-0.5 w-fit">
              <button
                type="button"
                onClick={() => setTargetType("chain")}
                className={`px-3 py-1 text-sm rounded transition-colors ${
                  targetType === "chain"
                    ? "bg-primary text-primary-foreground"
                    : "hover:bg-muted text-muted-foreground"
                }`}
              >
                Chain
              </button>
              <button
                type="button"
                onClick={() => setTargetType("template")}
                className={`px-3 py-1 text-sm rounded transition-colors ${
                  targetType === "template"
                    ? "bg-primary text-primary-foreground"
                    : "hover:bg-muted text-muted-foreground"
                }`}
              >
                Template
              </button>
            </div>

            {targetType === "chain" && (
              <select
                value={chainRef}
                onChange={(e) => setChainRef(e.target.value)}
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                <option value="">Select a chain...</option>
                {chains.map((c) => (
                  <option key={c.metadata.name} value={c.metadata.name}>
                    {c.spec.displayName || c.metadata.name}
                  </option>
                ))}
              </select>
            )}

            {targetType === "template" && (
              <select
                value={templateRef}
                onChange={(e) => setTemplateRef(e.target.value)}
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                <option value="">Select a template...</option>
                {templates.map((t) => (
                  <option key={t.metadata.name} value={t.metadata.name}>
                    {t.spec.displayName || t.metadata.name}
                  </option>
                ))}
              </select>
            )}
          </div>

          <div className="flex gap-2 pt-2">
            <Button type="submit" size="sm" disabled={submitDisabled}>
              {submitting ? "Creating..." : "Create Schedule"}
            </Button>
            <Button type="button" size="sm" variant="ghost" onClick={() => navigate("/schedules")}>
              Cancel
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
