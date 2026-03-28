import { useState, useEffect } from "react";
import { useNavigate, Link } from "react-router-dom";
import { toast } from "sonner";
import { apiFetch } from "../hooks/apiFetch";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";

interface ProjectSummary {
  name: string;
  displayName?: string;
}

export default function TemplateNewView() {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [projectRef, setProjectRef] = useState("");
  const [prompt, setPrompt] = useState("");
  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    apiFetch("/api/v1/projects")
      .then((r) => r.ok ? r.json() : [])
      .then((data) => setProjects(Array.isArray(data) ? data : (data.items || [])))
      .catch(() => {});
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name || submitting) return;
    setSubmitting(true);
    try {
      const body: Record<string, string> = { name };
      if (displayName) body.displayName = displayName;
      if (description) body.description = description;
      if (projectRef) body.projectRef = projectRef;
      if (prompt) body.prompt = prompt;

      const resp = await apiFetch("/api/v1/templates", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (resp.ok) {
        toast.success("Template created");
        navigate("/templates");
      } else {
        const data = await resp.json();
        toast.error(data.error || "Failed to create template");
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <Link to="/templates" className="text-xs text-muted-foreground hover:text-foreground">Templates</Link>
        <span className="text-xs text-muted-foreground">/</span>
        <span className="font-semibold text-sm">New Template</span>
      </div>

      <div className="flex-1 overflow-y-auto overscroll-none">
        <form onSubmit={handleSubmit} className="max-w-2xl p-4 space-y-4">
          <div className="space-y-1">
            <label className="text-sm font-medium">Name <span className="text-destructive">*</span></label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-template"
              className="text-sm font-mono"
            />
          </div>

          <div className="space-y-1">
            <label className="text-sm font-medium">Display Name</label>
            <Input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="My Template"
              className="text-sm"
            />
          </div>

          <div className="space-y-1">
            <label className="text-sm font-medium">Description</label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Brief description"
              className="text-sm"
            />
          </div>

          <div className="space-y-1">
            <label className="text-sm font-medium">Project</label>
            <select
              value={projectRef}
              onChange={(e) => setProjectRef(e.target.value)}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
            >
              <option value="">— none —</option>
              {projects.map((p) => (
                <option key={p.name} value={p.name}>
                  {p.displayName || p.name}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-1">
            <label className="text-sm font-medium">Prompt</label>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Enter prompt..."
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 resize-y min-h-[120px]"
            />
          </div>

          <div className="flex gap-2">
            <Button type="submit" size="sm" disabled={!name || submitting}>
              {submitting ? "Creating..." : "Create template"}
            </Button>
            <Button type="button" size="sm" variant="ghost" onClick={() => navigate("/templates")}>
              Cancel
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
