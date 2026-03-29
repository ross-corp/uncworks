import { useState, useEffect, useCallback } from "react";
import { useNavigate, useLocation, Link } from "react-router-dom";
import { toast } from "sonner";
import cronstrue from "cronstrue";
import { apiFetch } from "../hooks/apiFetch";
import { formatAge, formatRelative } from "../lib/format";
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

interface ScheduleSummary {
  metadata: { name: string; creationTimestamp: string };
  spec: {
    displayName?: string;
    cron: string;
    suspend: boolean;
    chainRef?: string;
    templateRef?: string;
  };
  status: {
    lastResult?: string;
    lastRunId?: string;
    nextScheduleTime?: string;
  };
}

export default function ScheduleListView() {
  const navigate = useNavigate();
  const location = useLocation();
  const [schedules, setSchedules] = useState<ScheduleSummary[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    try {
      const resp = await apiFetch("/api/v1/schedules");
      if (resp.ok) setSchedules(await resp.json());
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to load schedules");
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

  // Keyboard shortcut: n → navigate to new schedule form
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (location.pathname !== "/schedules") return;
      const tag = (e.target as HTMLElement).tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
      if (e.key === "n") navigate("/schedules/new");
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [location.pathname, navigate]);

  async function toggleSuspend(name: string, suspended: boolean) {
    try {
      const resp = await apiFetch(`/api/v1/schedules/${name}/${suspended ? "resume" : "suspend"}`, { method: "POST" });
      if (resp.ok) {
        fetchData();
      } else {
        const data = await resp.json().catch(() => ({}));
        toast.error((data as { error?: string }).error || "Failed to update schedule");
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to update schedule");
    }
  }

  async function deleteSchedule(name: string) {
    try {
      const resp = await apiFetch(`/api/v1/schedules/${name}`, { method: "DELETE" });
      if (resp.ok) {
        toast.success("Schedule deleted");
        fetchData();
      } else {
        const data = await resp.json().catch(() => ({}));
        toast.error((data as { error?: string }).error || "Failed to delete schedule");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete schedule");
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <span className="font-semibold flex-1">Schedules</span>
        <Badge variant="secondary" className="text-xs">{schedules.length}</Badge>
      </div>

      <div className="flex-1 overflow-y-auto overscroll-none">
        {loading && schedules.length === 0 && (
          <div className="flex h-full items-center justify-center">
            <Spinner className="text-muted-foreground" />
          </div>
        )}
        {!loading && schedules.length === 0 && (
          <Empty className="h-full border-0">
            <EmptyHeader>
              <EmptyTitle>No schedules configured</EmptyTitle>
              <EmptyDescription>Schedules run chains or templates automatically on a cron expression.</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <span className="text-xs text-muted-foreground">Press <kbd className="font-mono">n</kbd> to create</span>
            </EmptyContent>
          </Empty>
        )}

        {schedules.map((s) => (
          <div
            key={s.metadata.name}
            className="flex items-center gap-3 px-4 py-2.5 border-b border-border/40 hover:bg-muted/30 transition-colors"
          >
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <Link
                  to={`/schedules/${s.metadata.name}`}
                  className="font-medium hover:underline"
                >
                  {s.spec.displayName || s.metadata.name}
                </Link>
                <span
                  className="text-xs text-muted-foreground font-mono bg-muted px-1.5 py-0.5 rounded cursor-help"
                  title={(() => { try { return cronstrue.toString(s.spec.cron); } catch { return s.spec.cron; } })()}
                >
                  {s.spec.cron}
                </span>
                {s.spec.suspend && <Badge variant="secondary" className="text-xs">suspended</Badge>}
              </div>
              <div className="text-xs text-muted-foreground mt-0.5">
                {s.spec.chainRef ? `Chain: ${s.spec.chainRef}` : `Template: ${s.spec.templateRef}`}
                {s.status.lastResult && ` -- last: ${s.status.lastResult}`}
                {s.status.nextScheduleTime && ` -- next: ${formatRelative(s.status.nextScheduleTime)}`}
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Button
                size="sm"
                variant="ghost"
                onClick={() => toggleSuspend(s.metadata.name, s.spec.suspend)}
              >
                {s.spec.suspend ? "resume" : "suspend"}
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="text-destructive hover:text-destructive"
                onClick={() => deleteSchedule(s.metadata.name)}
              >
                delete
              </Button>
              <span className="text-xs text-muted-foreground">{formatAge(s.metadata.creationTimestamp)}</span>
            </div>
          </div>
        ))}
      </div>
      <div className="border-t px-4 py-1.5 text-xs text-muted-foreground">
        n new
      </div>
    </div>
  );
}
