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
        <LogViewerInner lines={lines} streaming={streaming} />
      </Suspense>
    </div>
  );
}
