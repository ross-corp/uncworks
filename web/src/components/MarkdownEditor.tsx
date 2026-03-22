import { Suspense, lazy } from "react";
import { useThemeNew } from "../hooks/useThemeNew";

const MonacoEditor = lazy(() => import("@monaco-editor/react"));

interface MarkdownEditorProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  minHeight?: string;
  autoFocus?: boolean;
}

export default function MarkdownEditor({
  value,
  onChange,
  placeholder,
  minHeight = "200px",
  autoFocus = false,
}: MarkdownEditorProps) {
  const { resolvedTheme } = useThemeNew();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";

  return (
    <div className="border" style={{ minHeight }}>
      <Suspense
        fallback={
          <div
            className="flex items-center justify-center text-sm text-muted-foreground"
            style={{ minHeight }}
          >
            Loading editor...
          </div>
        }
      >
        <MonacoEditor
          height={minHeight}
          language="markdown"
          theme={monacoTheme}
          value={value}
          onChange={(v) => onChange(v ?? "")}
          options={{
            minimap: { enabled: false },
            wordWrap: "on",
            lineNumbers: "off",
            scrollBeyondLastLine: false,
            fontSize: 13,
            fontFamily: "'IoskeleyMono', monospace",
            padding: { top: 8, bottom: 8 },
            renderLineHighlight: "none",
            overviewRulerLanes: 0,
            hideCursorInOverviewRuler: true,
            scrollbar: { vertical: "auto", horizontal: "hidden" },
            automaticLayout: true,
            placeholder,
          }}
          onMount={(editor) => {
            if (autoFocus) editor.focus();
          }}
        />
      </Suspense>
    </div>
  );
}
