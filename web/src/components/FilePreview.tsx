import { Suspense, lazy } from "react";

const MonacoEditor = lazy(() => import("@monaco-editor/react"));

const EXT_TO_LANG: Record<string, string> = {
  ts: "typescript",
  tsx: "typescript",
  js: "javascript",
  jsx: "javascript",
  go: "go",
  py: "python",
  rs: "rust",
  json: "json",
  yaml: "yaml",
  yml: "yaml",
  toml: "toml",
  md: "markdown",
  css: "css",
  html: "html",
  sql: "sql",
  sh: "shell",
  bash: "shell",
  dockerfile: "dockerfile",
  proto: "protobuf",
  xml: "xml",
  svg: "xml",
};

function detectLanguage(path: string): string {
  const filename = path.split("/").pop() ?? "";
  const lower = filename.toLowerCase();

  // Handle special filenames
  if (lower === "dockerfile") return "dockerfile";
  if (lower === "makefile") return "makefile";

  const ext = lower.split(".").pop() ?? "";
  return EXT_TO_LANG[ext] ?? "plaintext";
}

export default function FilePreview({
  path,
  content,
}: {
  path: string;
  content: string;
}) {
  const language = detectLanguage(path);

  return (
    <div className="flex h-full flex-col">
      {/* File path header */}
      <div className="border-b border-edge bg-surface-2 px-3 py-1.5">
        <span className="font-mono text-xs text-txt-tertiary truncate">
          {path}
        </span>
      </div>

      {/* Editor */}
      <div className="flex-1 overflow-hidden">
        <Suspense
          fallback={
            <div className="flex h-full items-center justify-center text-sm text-txt-tertiary">
              Loading editor...
            </div>
          }
        >
          <MonacoEditor
            height="100%"
            language={language}
            theme="vs-dark"
            value={content}
            options={{
              readOnly: true,
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
    </div>
  );
}
