import type { ReactNode } from "react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

export default function Layout({
  sidebar,
  children,
  detailPanel,
  onNewRun,
  searchQuery,
  onSearchChange,
  activeView = "runs",
}: {
  sidebar: ReactNode;
  children: ReactNode;
  detailPanel?: ReactNode;
  onNewRun: () => void;
  searchQuery?: string;
  onSearchChange?: (query: string) => void;
  activeView?: "runs" | "repos" | "events";
}) {
  return (
    <div className="flex h-screen">
      {sidebar}
      <div className="flex flex-1 overflow-hidden">
        <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
          <header className="flex items-center gap-3 border-b border-border px-6 py-3">
            {activeView === "repos" ? (
              <h2 className="text-sm font-semibold text-foreground fx-glow">Repositories</h2>
            ) : activeView === "events" ? (
              <h2 className="text-sm font-semibold text-foreground fx-glow">Events</h2>
            ) : (
              <>
                {onSearchChange != null && (
                  <Input
                    type="text"
                    data-testid="search-input"
                    value={searchQuery ?? ""}
                    onChange={(e) => onSearchChange(e.target.value)}
                    placeholder="Search agent runs..."
                    className="flex-1 text-sm"
                  />
                )}
                <Button data-testid="new-run-button" onClick={onNewRun} className="ml-auto">
                  + New Agent Run
                </Button>
              </>
            )}
          </header>
          <main className="flex-1 overflow-y-auto fx-scanlines">{children}</main>
        </div>
        <div
          className={`shrink-0 overflow-hidden transition-all duration-200 ${
            detailPanel ? "w-[480px]" : "w-0"
          }`}
        >
          {detailPanel}
        </div>
      </div>
    </div>
  );
}
