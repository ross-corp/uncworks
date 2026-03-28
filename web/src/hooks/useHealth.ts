// useHealth.ts — Poll UNCWORKS dependency health from the Wails backend.
import { useState, useEffect, useCallback } from "react";
import { isWails } from "../lib/wails-env";

export type HealthStatus = "ok" | "degraded" | "down" | "unknown";

export interface HealthComponent {
  name: string;
  label: string;
  status: HealthStatus;
  message: string;
}

export interface HealthReport {
  overall: HealthStatus;
  components: HealthComponent[];
}

const EMPTY: HealthReport = { overall: "unknown", components: [] };

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const go = () => (window as any).go?.main?.App;

export function useHealth(intervalMs = 15_000) {
  const wails = isWails();
  const [report, setReport] = useState<HealthReport>(EMPTY);
  const [loading, setLoading] = useState(false);

  const check = useCallback(async () => {
    if (!wails) return;
    setLoading(true);
    try {
      const r: HealthReport = await go().HealthCheck();
      setReport(r);
    } catch {
      setReport({ overall: "down", components: [] });
    } finally {
      setLoading(false);
    }
  }, [wails]);

  useEffect(() => {
    check();
    const t = setInterval(check, intervalMs);
    return () => clearInterval(t);
  }, [check, intervalMs]);

  return { report, loading, refresh: check };
}

export function statusColor(s: HealthStatus): string {
  switch (s) {
    case "ok":       return "bg-green-500";
    case "degraded": return "bg-yellow-500";
    case "down":     return "bg-red-500";
    default:         return "bg-muted-foreground";
  }
}

export function statusLabel(s: HealthStatus): string {
  switch (s) {
    case "ok":       return "Healthy";
    case "degraded": return "Degraded";
    case "down":     return "Down";
    default:         return "Unknown";
  }
}
