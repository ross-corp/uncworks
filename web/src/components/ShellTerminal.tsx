import { Suspense, lazy } from "react";

const ShellTerminalInner = lazy(() => import("./ShellTerminalInner"));

export default function ShellTerminal({ runId }: { runId: string }) {
  return (
    <div
      data-testid="shell-terminal"
      className="overflow-hidden rounded border border-edge bg-surface-1"
      style={{ height: "100%" }}
    >
      <Suspense
        fallback={
          <div className="flex h-full items-center justify-center text-sm text-txt-tertiary">
            Loading terminal...
          </div>
        }
      >
        <ShellTerminalInner runId={runId} />
      </Suspense>
    </div>
  );
}
