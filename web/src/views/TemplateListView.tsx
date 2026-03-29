import { useState, useEffect, useCallback } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { toast } from "sonner";
import { apiFetch } from "../hooks/apiFetch";
import { formatAge } from "../lib/format";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Spinner } from "../components/ui/spinner";
import {
  Empty,
  EmptyHeader,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
} from "../components/ui/empty";

interface TemplateSummary {
  metadata: { name: string; creationTimestamp: string };
  spec: { displayName?: string; description?: string; projectRef?: string; prompt?: string };
  status?: { runCount?: number };
}

export default function TemplateListView() {
  const navigate = useNavigate();
  const location = useLocation();
  const [templates, setTemplates] = useState<TemplateSummary[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    try {
      const resp = await apiFetch("/api/v1/templates");
      if (resp.ok) setTemplates(await resp.json());
      else toast.error("Failed to load templates");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to load templates");
    } finally { setLoading(false); }
  }, []);

  useEffect(() => {
    let cancelled = false;
    fetchData();
    const i = setInterval(() => {
      if (!cancelled) fetchData();
    }, 10000);
    return () => {
      cancelled = true;
      clearInterval(i);
    };
  }, [fetchData]);

  // Keyboard shortcut: n → navigate to new template form
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (location.pathname !== "/templates") return;
      const tag = (e.target as HTMLElement).tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
      if (e.key === "n") navigate("/templates/new");
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [location.pathname, navigate]);

  async function deleteTemplate(name: string) {
    if (!window.confirm(`Delete template "${name}"? This cannot be undone.`)) return;
    try {
      const resp = await apiFetch(`/api/v1/templates/${name}`, { method: "DELETE" });
      if (resp.status === 409) {
        const data = await resp.json();
        toast.error(data.error);
        return;
      }
      if (resp.ok) {
        toast.success("Template deleted");
        fetchData();
      } else {
        const data = await resp.json().catch(() => ({}));
        toast.error((data as { error?: string }).error || "Failed to delete template");
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to delete template");
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <span className="font-semibold flex-1">Templates</span>
      </div>

      <div className="flex-1 overflow-y-auto overscroll-none">
        {loading && (
          <div className="flex h-full items-center justify-center">
            <Spinner className="text-muted-foreground" />
          </div>
        )}

        {!loading && templates.length === 0 && (
          <Empty className="h-full border-0">
            <EmptyHeader>
              <EmptyTitle>No templates yet</EmptyTitle>
              <EmptyDescription>Templates are reusable run configurations you can compose into chains or trigger on a schedule.</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <span className="text-xs text-muted-foreground">Press <kbd className="font-mono">n</kbd> to create</span>
            </EmptyContent>
          </Empty>
        )}

        {!loading && templates.map((t) => {
          const runCount = t.status?.runCount ?? 0;
          return (
            <div
              key={t.metadata.name}
              className="px-4 py-2.5 border-b border-border/40 hover:bg-muted/30 flex items-center gap-3 transition-colors"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">{t.spec.displayName || t.metadata.name}</span>
                  {t.spec.projectRef && (
                    <Badge variant="secondary" className="text-xs">{t.spec.projectRef}</Badge>
                  )}
                </div>
                {t.spec.description && (
                  <div className="text-xs text-muted-foreground mt-0.5">{t.spec.description}</div>
                )}
              </div>
              <div className="flex items-center gap-2 shrink-0">
                {runCount > 0 && (
                  <Badge variant="secondary">{runCount} run{runCount !== 1 ? "s" : ""}</Badge>
                )}
                <span className="text-xs text-muted-foreground">{formatAge(t.metadata.creationTimestamp)}</span>
                <Button
                  size="sm"
                  variant="ghost"
                  className="text-xs px-2 text-destructive hover:text-destructive"
                  onClick={() => deleteTemplate(t.metadata.name)}
                >
                  delete
                </Button>
              </div>
            </div>
          );
        })}
      </div>
      <div className="border-t px-4 py-1.5 text-xs text-muted-foreground">
        n new
      </div>
    </div>
  );
}
