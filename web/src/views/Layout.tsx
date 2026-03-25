import { Outlet } from "react-router-dom";
import GlobalNav from "../components/GlobalNav";
import ErrorBoundary from "../components/ErrorBoundary";
import { CopilotContextProvider } from "../hooks/useCopilotContext";
import CopilotPanel from "../components/CopilotPanel";

export default function Layout() {
  return (
    <CopilotContextProvider>
      <div className="h-screen w-screen overflow-hidden bg-background text-foreground font-mono text-sm flex flex-row">
        <GlobalNav />
        <main className="flex-1 min-w-0 flex flex-col h-screen overflow-hidden">
          <ErrorBoundary>
            <Outlet />
          </ErrorBoundary>
        </main>
        <CopilotPanel />
      </div>
    </CopilotContextProvider>
  );
}
