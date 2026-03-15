import { Suspense, lazy } from "react";

const MonacoEditor = lazy(() => import("@monaco-editor/react"));

export default function SpecEditor({
  value,
  onChange,
  readOnly = false,
  height = "300px",
}: {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  height?: string;
}) {
  return (
    <div
      data-testid="spec-editor"
      className="overflow-hidden border border-border bg-card"
      style={{ height }}
    >
      <Suspense
        fallback={
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
            Loading editor...
          </div>
        }
      >
        <MonacoEditor
          height={height}
          language="markdown"
          theme="vs-dark"
          value={value}
          onChange={(v) => onChange?.(v ?? "")}
          options={{
            readOnly,
            minimap: { enabled: false },
            wordWrap: "on",
            lineNumbers: "on",
            scrollBeyondLastLine: false,
            fontSize: 13,
            fontFamily: "'IoskeleyMono', monospace",
            padding: { top: 8 },
          }}
        />
      </Suspense>
    </div>
  );
}
