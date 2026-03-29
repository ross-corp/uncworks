import { Outlet } from "react-router-dom";
import { useState, useEffect } from "react";
import { toast } from "sonner";
import GlobalNav from "../components/GlobalNav";
import ErrorBoundary from "../components/ErrorBoundary";
import { CopilotContextProvider } from "../hooks/useCopilotContext";
import { useCopilotContextValue } from "../hooks/useCopilotContext";
import CopilotBottomPanel from "../components/CopilotBottomPanel";
import { HealthProvider } from "../hooks/useHealthContext";
import { SettingsProvider, useSettings } from "../hooks/useSettings";
import { isWails } from "../lib/wails-env";
import SetupWizardModal from "../components/SetupWizard";

// TITLEBAR_H: height of the macOS hidden-inset title bar.
// The drag region spans the full window width at this height so the window
// is draggable everywhere above the content, and no content is clipped.
const TITLEBAR_H = 36;

function LayoutInner() {
  const { open, panelHeight } = useCopilotContextValue();
  const { configStatus, reload, loading } = useSettings();
  const wails = isWails();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [wizardChecked, setWizardChecked] = useState(false);

  // Listen for local-channel hot-reload signal from the Go backend.
  useEffect(() => {
    const handler = () => toast.info("New local build detected — reloading…");
    window.addEventListener("uncworks:local-reload", handler);
    return () => window.removeEventListener("uncworks:local-reload", handler);
  }, []);

  // Auto-show wizard on first launch if not yet complete.
  // Wait for settings to finish loading so we don't open wizard
  // immediately with SETTINGS_DEFAULTS (wizardComplete=false).
  useEffect(() => {
    if (!wails || wizardChecked || loading) return;
    setWizardChecked(true);
    if (!configStatus.wizardComplete) {
      setWizardOpen(true);
    }
  }, [wails, configStatus, wizardChecked, loading]);

  return (
    <div className="h-screen w-screen overflow-hidden bg-background text-foreground font-mono text-sm flex flex-col">
      {wizardOpen && <SetupWizardModal onClose={() => { setWizardOpen(false); reload(); }} />}
      {/* Full-width macOS title bar drag zone — only rendered in Wails desktop */}
      {wails && (
        <div
          className="shrink-0 w-full border-b border-border/50"
          style={{ height: TITLEBAR_H, WebkitAppRegion: "drag" } as React.CSSProperties}
        />
      )}

      {/* Content below the title bar */}
      <div
        className="flex flex-row flex-1 min-h-0 overflow-hidden"
        style={{ paddingBottom: open ? panelHeight : 0 }}
      >
        <GlobalNav />
        <main className="flex-1 min-w-0 flex flex-col overflow-hidden">
          <ErrorBoundary>
            <Outlet />
          </ErrorBoundary>
        </main>
        <CopilotBottomPanel />
      </div>
    </div>
  );
}

export default function Layout() {
  return (
    <SettingsProvider>
      <HealthProvider>
        <CopilotContextProvider>
          <LayoutInner />
        </CopilotContextProvider>
      </HealthProvider>
    </SettingsProvider>
  );
}
