import { Outlet } from "react-router-dom";
import GlobalNav from "../components/GlobalNav";
import ErrorBoundary from "../components/ErrorBoundary";
import { CopilotContextProvider } from "../hooks/useCopilotContext";
import { useCopilotContextValue } from "../hooks/useCopilotContext";
import CopilotBottomPanel from "../components/CopilotBottomPanel";
import { HealthProvider } from "../hooks/useHealthContext";
import { SettingsProvider } from "../hooks/useSettings";

// TITLEBAR_H: height of the macOS hidden-inset title bar.
// The drag region spans the full window width at this height so the window
// is draggable everywhere above the content, and no content is clipped.
const TITLEBAR_H = 36;

function LayoutInner() {
  const { open, panelHeight } = useCopilotContextValue();
  return (
    <div className="h-screen w-screen overflow-hidden bg-background text-foreground font-mono text-sm flex flex-col">
      {/* Full-width macOS title bar drag zone */}
      <div
        className="shrink-0 w-full border-b border-border/50"
        style={{ height: TITLEBAR_H, WebkitAppRegion: "drag" } as React.CSSProperties}
      />

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
