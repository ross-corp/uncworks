import { useState, useEffect, lazy, Suspense } from "react";
import type { SpanDiff } from "../types/agent-run";

// ============================================================
// DiffViewer v2 — Monaco-based side-by-side diff editor
// Lazy-loads Monaco on first open, MU-TH-UR styled container.
// ============================================================

/** Lazy-loaded Monaco diff editor component */
const MonacoDiffEditor = lazy(() => import("./MonacoDiffEditor"));

/** Loading skeleton in MU-TH-UR style */
function DiffLoadingSkeleton() {
  return (
    <div
      className="flex items-center justify-center py-16"
      style={{
        background: "var(--muthr-bg)",
        fontFamily: "var(--muthr-font)",
      }}
    >
      <span
        className="text-sm uppercase tracking-widest"
        style={{ color: "var(--muthr-green)" }}
      >
        Loading diff
        <span className="muthr-cursor">_</span>
      </span>
    </div>
  );
}

/** Extract language ID from file path */
function getLanguageFromPath(filePath: string): string {
  const ext = filePath.split(".").pop()?.toLowerCase() ?? "";
  const langMap: Record<string, string> = {
    ts: "typescript",
    tsx: "typescript",
    js: "javascript",
    jsx: "javascript",
    go: "go",
    py: "python",
    rs: "rust",
    java: "java",
    json: "json",
    yaml: "yaml",
    yml: "yaml",
    md: "markdown",
    css: "css",
    scss: "scss",
    html: "html",
    xml: "xml",
    sql: "sql",
    sh: "shell",
    bash: "shell",
    zsh: "shell",
    proto: "protobuf",
    toml: "ini",
    dockerfile: "dockerfile",
  };
  return langMap[ext] ?? "plaintext";
}

/** Parse a unified diff patch into before/after content */
function parsePatch(patch: string): { before: string; after: string } {
  const lines = patch.split("\n");
  const beforeLines: string[] = [];
  const afterLines: string[] = [];

  for (const line of lines) {
    if (line.startsWith("---") || line.startsWith("+++")) {
      continue;
    }
    if (line.startsWith("@@")) {
      continue;
    }
    if (line.startsWith("-")) {
      beforeLines.push(line.slice(1));
    } else if (line.startsWith("+")) {
      afterLines.push(line.slice(1));
    } else {
      // Context line (starts with space or is empty)
      const content = line.startsWith(" ") ? line.slice(1) : line;
      beforeLines.push(content);
      afterLines.push(content);
    }
  }

  return {
    before: beforeLines.join("\n"),
    after: afterLines.join("\n"),
  };
}

/** Count added/removed lines in a patch */
function countChanges(patch: string): { added: number; removed: number } {
  const lines = patch.split("\n");
  let added = 0;
  let removed = 0;
  for (const line of lines) {
    if (line.startsWith("+") && !line.startsWith("+++")) added++;
    if (line.startsWith("-") && !line.startsWith("---")) removed++;
  }
  return { added, removed };
}

export default function DiffViewer({
  diff,
  modal,
}: {
  diff: SpanDiff;
  modal?: boolean;
}) {
  const [selectedFileIndex, setSelectedFileIndex] = useState(0);
  const [inlineMode, setInlineMode] = useState(false);

  // Reset selection when diff changes
  useEffect(() => {
    setSelectedFileIndex(0);
  }, [diff]);

  if (!diff.files || diff.files.length === 0) {
    return (
      <div
        className="flex items-center justify-center py-8"
        style={{
          background: "var(--muthr-bg)",
          fontFamily: "var(--muthr-font)",
        }}
      >
        <span
          className="text-[11px] uppercase tracking-widest"
          style={{ color: "var(--muthr-dim-green)" }}
        >
          No file changes in this span
        </span>
      </div>
    );
  }

  const selectedFile = diff.files[selectedFileIndex];
  const { before, after } = parsePatch(selectedFile.patch);
  const language = getLanguageFromPath(selectedFile.path);

  return (
    <div
      className={`flex flex-col overflow-hidden ${modal ? "fixed inset-4 z-50" : ""}`}
      style={{
        background: "var(--muthr-bg)",
        fontFamily: "var(--muthr-font)",
        border: modal ? "1px solid var(--muthr-dim-green)" : undefined,
      }}
    >
      {/* Modal backdrop */}
      {modal && (
        <div className="fixed inset-0 bg-black/80 -z-10" />
      )}

      {/* Header */}
      <div
        className="flex items-center justify-between px-3 py-2 border-b"
        style={{ borderColor: "var(--muthr-dim-green)" }}
      >
        <span
          className="text-[11px] uppercase tracking-wider truncate"
          style={{ color: "var(--muthr-green)" }}
        >
          {selectedFile.path}
        </span>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setInlineMode(!inlineMode)}
            className="px-2 py-0.5 text-[10px] uppercase tracking-widest border transition-colors"
            style={{
              borderColor: inlineMode ? "var(--muthr-green)" : "var(--muthr-dim-green)",
              color: inlineMode ? "var(--muthr-green)" : "var(--muthr-dim-green)",
              background: "transparent",
            }}
          >
            {inlineMode ? "Inline" : "Side-by-Side"}
          </button>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* File list sidebar (multi-file) */}
        {diff.files.length > 1 && (
          <div
            className="flex flex-col w-48 flex-shrink-0 border-r overflow-y-auto"
            style={{ borderColor: "var(--muthr-dim-green)" }}
          >
            {diff.files.map((file, i) => {
              const changes = countChanges(file.patch);
              const isActive = i === selectedFileIndex;
              return (
                <button
                  key={i}
                  onClick={() => setSelectedFileIndex(i)}
                  className="flex flex-col items-start px-2 py-1.5 text-left transition-colors"
                  style={{
                    background: isActive ? "rgba(0, 255, 65, 0.08)" : "transparent",
                    borderLeft: isActive ? "2px solid var(--muthr-green)" : "2px solid transparent",
                  }}
                >
                  <span
                    className="text-[10px] truncate w-full"
                    style={{
                      color: isActive ? "var(--muthr-green)" : "var(--muthr-dim-green)",
                      fontFamily: "var(--muthr-font)",
                    }}
                  >
                    {file.path.split("/").pop()}
                  </span>
                  <span className="flex items-center gap-2 text-[9px]">
                    <span style={{ color: "var(--muthr-green)" }}>+{changes.added}</span>
                    <span style={{ color: "var(--muthr-amber)" }}>-{changes.removed}</span>
                  </span>
                </button>
              );
            })}
          </div>
        )}

        {/* Monaco diff editor */}
        <div className="flex-1 overflow-hidden">
          <Suspense fallback={<DiffLoadingSkeleton />}>
            <MonacoDiffEditor
              original={before}
              modified={after}
              language={language}
              inlineMode={inlineMode}
            />
          </Suspense>
        </div>
      </div>
    </div>
  );
}
