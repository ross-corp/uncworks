import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { apiFetch } from "../hooks/apiFetch";
import { formatAge } from "../lib/format";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";

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
  const [schedules, setSchedules] = useState<ScheduleSummary[]>([]);
  const [loading, setLoading] = useState(true);

  const fetch = useCallback(async () => {
    try {
      const resp = await apiFetch("/api/v1/schedules");
      if (resp.ok) setSchedules(await resp.json());
    } catch { /* silent */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetch(); const i = setInterval(fetch, 10000); return () => clearInterval(i); }, [fetch]);

  async function toggleSuspend(name: string, suspended: boolean) {
    await apiFetch(`/api/v1/schedules/${name}/${suspended ? "resume" : "suspend"}`, { method: "POST" });
    fetch();
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b px-4 py-2">
        <div className="flex items-center gap-3">
          <span className="font-semibold">Schedules</span>
          <span className="text-muted-foreground text-xs">({schedules.length})</span>
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="ghost" className="h-6 text-[11px]" onClick={() => navigate("/")}>
            Runs
          </Button>
          <Button size="sm" variant="ghost" className="h-6 text-[11px]" onClick={() => navigate("/chains")}>
            Chains
          </Button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {loading && schedules.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground">Loading...</div>
        )}
        {!loading && schedules.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground">
            No schedules configured
          </div>
        )}

        {schedules.map((s) => (
          <div
            key={s.metadata.name}
            className="flex items-center gap-3 px-4 py-3 border-b border-border/50 hover:bg-muted/30 transition-colors"
          >
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-medium">{s.spec.displayName || s.metadata.name}</span>
                <span className="text-xs text-muted-foreground font-mono bg-muted px-1.5 py-0.5 rounded">
                  {s.spec.cron}
                </span>
                {s.spec.suspend && <Badge variant="secondary" className="text-[10px]">suspended</Badge>}
              </div>
              <div className="text-xs text-muted-foreground mt-0.5">
                {s.spec.chainRef ? `Chain: ${s.spec.chainRef}` : `Template: ${s.spec.templateRef}`}
                {s.status.lastResult && ` -- last: ${s.status.lastResult}`}
                {s.status.nextScheduleTime && ` -- next: ${new Date(s.status.nextScheduleTime).toLocaleString()}`}
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <button
                onClick={() => toggleSuspend(s.metadata.name, s.spec.suspend)}
                className="text-[11px] px-2 py-0.5 rounded text-muted-foreground hover:text-foreground transition-colors"
              >
                {s.spec.suspend ? "resume" : "suspend"}
              </button>
              <span className="text-[11px] text-muted-foreground">{formatAge(s.metadata.creationTimestamp)}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
