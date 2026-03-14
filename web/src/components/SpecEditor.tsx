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
      className="overflow-hidden rounded border border-edge bg-surface-1"
      style={{ height }}
    >
      <Suspense
        fallback={
          <div className="flex h-full items-center justify-center text-sm text-txt-tertiary">
            Loading editor...
          </div>
        }
      >
        <MonacoEditor
          height="100%"
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
            fontFamily: "'JetBrains Mono', monospace",
            padding: { top: 8 },
          }}
        />
      </Suspense>
    </div>
  );
}
