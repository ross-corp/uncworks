import { Suspense, lazy } from "react";

const ShellTerminalInner = lazy(() => import("./ShellTerminalInner"));

export default function ShellTerminal({ runId }: { runId: string }) {
  return (
    <div
      data-testid="shell-terminal"
      className="overflow-hidden border border-border bg-background fx-scanlines"
      style={{ height: "100%" }}
    >
      <Suspense
        fallback={
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
            Loading terminal...
          </div>
        }
      >
        <ShellTerminalInner runId={runId} />
      </Suspense>
    </div>
  );
}
