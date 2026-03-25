import { useState, useEffect, useCallback } from "react";
import { useParams, Link } from "react-router-dom";
import { toast } from "sonner";
import cronstrue from "cronstrue";
import { apiFetch } from "../hooks/apiFetch";
import { formatAge } from "../lib/format";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import RunStatusBadge from "../components/RunStatusBadge";
import type { AgentRun } from "../types/agent-run";

interface ScheduleDetail {
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

export default function ScheduleDetailView() {
  const { name } = useParams<{ name: string }>();
  const [schedule, setSchedule] = useState<ScheduleDetail | null>(null);
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [editCron, setEditCron] = useState("");
  const [savingCron, setSavingCron] = useState(false);
  const [toggling, setToggling] = useState(false);

  const fetchSchedule = useCallback(async () => {
    if (!name) return;
    try {
      const resp = await apiFetch(`/api/v1/schedules/${name}`);
      if (resp.ok) {
        const data: ScheduleDetail = await resp.json();
        setSchedule(data);
        setEditCron(data.spec.cron);
      }
    } catch (e) {
      toast.error(`Failed to load schedule: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setLoading(false);
    }
  }, [name]);

  useEffect(() => {
    let cancelled = false;

    const fetchAll = async () => {
      if (!name) return;

      try {
        const resp = await apiFetch(`/api/v1/schedules/${name}`);
        if (resp.ok) {
          const data: ScheduleDetail = await resp.json();
          if (!cancelled) {
            setSchedule(data);
            setEditCron(data.spec.cron);
          }
        }
      } catch (e) {
        if (!cancelled) toast.error(`Failed to load schedule: ${e instanceof Error ? e.message : String(e)}`);
      } finally {
        if (!cancelled) setLoading(false);
      }

      try {
        const resp = await apiFetch("/api/v1/runs");
        if (resp.ok) {
          const data: AgentRun[] = await resp.json();
          const filtered = data.filter(
            (r) =>
              (r.spec.tags && r.spec.tags.includes(`schedule:${name}`)) ||
              r.spec.feature === name
          );
          if (!cancelled) setRuns(filtered.slice(0, 10));
        }
      } catch (e) {
        if (!cancelled) toast.error(`Failed to load runs: ${e instanceof Error ? e.message : String(e)}`);
      }
    };

    fetchAll();
    const i = setInterval(() => {
      if (!cancelled) fetchAll();
    }, 10000);

    return () => {
      cancelled = true;
      clearInterval(i);
    };
  }, [name]);

  async function toggleSuspend() {
    if (!schedule) return;
    setToggling(true);
    try {
      const action = schedule.spec.suspend ? "resume" : "suspend";
      const resp = await apiFetch(`/api/v1/schedules/${name}/${action}`, { method: "POST" });
      if (resp.ok) {
        await fetchSchedule();
        toast.success(schedule.spec.suspend ? "Schedule resumed" : "Schedule suspended");
      }
    } catch (e) {
      toast.error(`Failed to toggle schedule: ${e instanceof Error ? e.message : String(e)}`);
    }
    setToggling(false);
  }

  async function saveCron() {
    if (!name || !editCron.trim()) return;
    setSavingCron(true);
    try {
      const resp = await apiFetch(`/api/v1/schedules/${name}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ cron: editCron.trim() }),
      });
      if (resp.ok) {
        await fetchSchedule();
        toast.success("Cron expression updated");
      } else {
        toast.error("Failed to update cron expression");
      }
    } catch (e) {
      toast.error(`Failed to save cron: ${e instanceof Error ? e.message : String(e)}`);
    }
    setSavingCron(false);
  }

  function humanCron(cron: string): string {
    try {
      return cronstrue.toString(cron);
    } catch {
      return cron;
    }
  }

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Loading schedule...
      </div>
    );
  }

  if (!schedule) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Schedule not found
      </div>
    );
  }

  const cronChanged = editCron !== schedule.spec.cron;

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="h-12 border-b flex items-center px-4 gap-2">
        <Link to="/schedules" className="text-xs text-muted-foreground hover:text-foreground transition-colors">Schedules</Link>
        <span className="text-muted-foreground">/</span>
        <span className="font-semibold flex-1">{schedule.spec.displayName || schedule.metadata.name}</span>
        {schedule.spec.suspend && (
          <Badge variant="secondary" className="text-xs">suspended</Badge>
        )}
        <Button
          size="sm"
          variant={schedule.spec.suspend ? "outline" : "ghost"}
          onClick={toggleSuspend}
          disabled={toggling}
        >
          {toggling ? "..." : schedule.spec.suspend ? "Resume" : "Suspend"}
        </Button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-6 max-w-2xl">
        {/* Cron info */}
        <div className="space-y-1">
          <div className="text-xs text-muted-foreground uppercase tracking-wider">Schedule</div>
          <div className="flex items-center gap-2">
            <span className="font-mono text-sm bg-muted px-2 py-0.5 rounded">{schedule.spec.cron}</span>
            <span className="text-sm text-muted-foreground">{humanCron(schedule.spec.cron)}</span>
          </div>
          {schedule.spec.chainRef && (
            <div className="text-xs text-muted-foreground">Chain: {schedule.spec.chainRef}</div>
          )}
          {schedule.spec.templateRef && (
            <div className="text-xs text-muted-foreground">Template: {schedule.spec.templateRef}</div>
          )}
          {schedule.status.nextScheduleTime && (
            <div className="text-xs text-muted-foreground">
              Next: {new Date(schedule.status.nextScheduleTime).toLocaleString()}
            </div>
          )}
        </div>

        {/* Cron editor */}
        <div className="space-y-2">
          <div className="text-xs text-muted-foreground uppercase tracking-wider">Edit Schedule</div>
          <div className="flex items-center gap-2">
            <Input
              className="h-8 text-sm font-mono w-48"
              value={editCron}
              onChange={(e) => setEditCron(e.target.value)}
              placeholder="* * * * *"
              onKeyDown={(e) => { if (e.key === "Enter" && cronChanged) saveCron(); }}
            />
            <Button
              size="sm"
              disabled={!cronChanged || savingCron}
              onClick={saveCron}
            >
              {savingCron ? "Saving..." : "Save"}
            </Button>
          </div>
          {editCron && (
            <div className="text-xs text-muted-foreground">{humanCron(editCron)}</div>
          )}
        </div>

        {/* Execution history */}
        <div className="space-y-2">
          <div className="text-xs text-muted-foreground uppercase tracking-wider">Recent Executions</div>
          {runs.length === 0 ? (
            <div className="text-sm text-muted-foreground py-4">No executions yet</div>
          ) : (
            <div className="border rounded divide-y divide-border/50">
              <div className="grid grid-cols-[1fr_auto_auto] gap-3 px-3 py-1.5 text-xs text-muted-foreground uppercase tracking-wider">
                <span>Run</span>
                <span>Status</span>
                <span>Age</span>
              </div>
              {runs.map((run) => (
                <Link
                  key={run.id}
                  to={`/run/${run.id}`}
                  className="grid grid-cols-[1fr_auto_auto] gap-3 px-3 py-2 hover:bg-muted/30 transition-colors items-center"
                >
                  <span className="text-sm truncate">{run.spec.displayName || run.name}</span>
                  <RunStatusBadge phase={run.status.phase} />
                  <span className="text-xs text-muted-foreground whitespace-nowrap">
                    {formatAge(run.createdAt)}
                  </span>
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
