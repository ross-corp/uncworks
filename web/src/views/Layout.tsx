import { Outlet } from "react-router-dom";
import GlobalNav from "../components/GlobalNav";
import ErrorBoundary from "../components/ErrorBoundary";
import { CopilotContextProvider } from "../hooks/useCopilotContext";
import { useCopilotContextValue } from "../hooks/useCopilotContext";
import CopilotBottomPanel from "../components/CopilotBottomPanel";

function LayoutInner() {
  const { open, panelHeight } = useCopilotContextValue();
  return (
    <div className="h-screen w-screen overflow-hidden bg-background text-foreground font-mono text-sm flex flex-row">
      <GlobalNav />
      <main
        className="flex-1 min-w-0 flex flex-col h-screen overflow-hidden"
        style={{ paddingBottom: open ? panelHeight : 0 }}
      >
        <ErrorBoundary>
          <Outlet />
        </ErrorBoundary>
      </main>
      <CopilotBottomPanel />
    </div>
  );
}

export default function Layout() {
  return (
    <CopilotContextProvider>
      <LayoutInner />
    </CopilotContextProvider>
  );
}
