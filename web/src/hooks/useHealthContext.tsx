// useHealthContext.tsx — App-level health gate: provides cluster/API readiness
// to any component that needs to block actions when dependencies are down.
import { createContext, useContext, ReactNode } from "react";
import { useHealth, type HealthReport, type HealthStatus } from "./useHealth";

interface HealthContextValue {
  report: HealthReport;
  loading: boolean;
  refresh: () => void;
  // Convenience gates
  clusterOk: boolean;   // kubernetes component is ok/degraded (cluster reachable)
  apiserverOk: boolean; // apiserver pod has ready replicas
  canSubmitRun: boolean; // both cluster + apiserver are up enough to submit
}

const HealthContext = createContext<HealthContextValue>({
  report: { overall: "unknown", components: [] },
  loading: false,
  refresh: () => {},
  clusterOk: true,
  apiserverOk: true,
  canSubmitRun: true,
});

function componentStatus(report: HealthReport, name: string): HealthStatus {
  return report.components.find(c => c.name === name)?.status ?? "unknown";
}

export function HealthProvider({ children }: { children: ReactNode }) {
  const { report, loading, refresh } = useHealth(15_000);

  const clusterStatus = componentStatus(report, "kubernetes");
  const apiserverStatus = componentStatus(report, "apiserver");

  const clusterOk = clusterStatus === "ok" || clusterStatus === "degraded" || clusterStatus === "unknown";
  const apiserverOk = apiserverStatus === "ok" || apiserverStatus === "degraded" || apiserverStatus === "unknown";
  const canSubmitRun = clusterOk && apiserverOk;

  return (
    <HealthContext.Provider value={{ report, loading, refresh, clusterOk, apiserverOk, canSubmitRun }}>
      {children}
    </HealthContext.Provider>
  );
}

export function useHealthContext() {
  return useContext(HealthContext);
}
