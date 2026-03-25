import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { apiFetch } from "../hooks/apiFetch";
import { formatAge } from "../lib/format";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";

interface TemplateSummary {
  metadata: { name: string; creationTimestamp: string };
  spec: { displayName?: string; description?: string; projectRef?: string; prompt?: string };
  status?: { runCount?: number };
}

export default function TemplateListView() {
  const navigate = useNavigate();
  const [templates, setTemplates] = useState<TemplateSummary[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    try {
      const resp = await apiFetch("/api/v1/templates");
      if (resp.ok) setTemplates(await resp.json());
    } catch { /* silent */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    fetchData();
    const i = setInterval(fetchData, 10000);
    return () => clearInterval(i);
  }, [fetchData]);

  async function deleteTemplate(name: string) {
    const resp = await apiFetch(`/api/v1/templates/${name}`, { method: "DELETE" });
    if (resp.status === 409) {
      const data = await resp.json();
      toast.error(data.error);
      return;
    }
    if (resp.ok) {
      toast.success("Template deleted");
      fetchData();
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <span className="font-semibold flex-1">Templates</span>
        <Button size="sm" onClick={() => navigate("/templates/new")}>+ new template</Button>
      </div>

      <div className="flex-1 overflow-y-auto">
        {loading && (
          <div className="flex h-full items-center justify-center text-muted-foreground">Loading...</div>
        )}

        {!loading && templates.length === 0 && (
          <div className="flex h-full flex-col items-center justify-center gap-3 text-muted-foreground">
            <span>No templates yet</span>
            <Button size="sm" onClick={() => navigate("/templates/new")}>+ new template</Button>
          </div>
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
                    <Badge variant="secondary" className="text-[10px]">{t.spec.projectRef}</Badge>
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
                  className="h-6 text-xs px-2 text-destructive hover:text-destructive"
                  onClick={() => deleteTemplate(t.metadata.name)}
                >
                  delete
                </Button>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
