import { Suspense, lazy } from "react";

const LogViewerInner = lazy(() => import("./LogViewerInner"));

export default function LogViewer({
  lines,
  streaming,
}: {
  lines: string[];
  streaming: boolean;
}) {
  return (
    <div
      data-testid="log-viewer"
      className="overflow-hidden border border-border bg-background fx-scanlines fx-noise"
      style={{ height: "100%" }}
    >
      <Suspense
        fallback={
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
            Loading terminal...
          </div>
        }
      >
        <LogViewerInner lines={lines} streaming={streaming} />
      </Suspense>
    </div>
  );
}
