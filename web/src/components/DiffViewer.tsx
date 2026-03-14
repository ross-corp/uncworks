import type { SpanDiff } from "../types/agent-run";

function classifyLine(line: string): string {
  if (line.startsWith("+") && !line.startsWith("+++")) return "text-green-400 bg-green-500/10";
  if (line.startsWith("-") && !line.startsWith("---")) return "text-red-400 bg-red-500/10";
  if (line.startsWith("@@")) return "text-blue-400";
  return "text-txt-secondary";
}

export default function DiffViewer({ diff }: { diff: SpanDiff }) {
  if (!diff.files || diff.files.length === 0) {
    return (
      <div className="flex items-center justify-center py-8 text-sm text-txt-tertiary">
        No file changes in this span
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 overflow-y-auto p-2">
      {diff.files.map((file, i) => (
        <div key={i} className="rounded border border-edge overflow-hidden">
          {/* File header */}
          <div className="border-b border-edge bg-surface-2 px-3 py-1.5">
            <span className="font-mono text-xs text-txt-secondary">{file.path}</span>
          </div>

          {/* Diff content */}
          <pre className="overflow-x-auto p-3 text-xs leading-5 font-mono bg-surface-0">
            {file.patch.split("\n").map((line, lineIdx) => (
              <div key={lineIdx} className={classifyLine(line)}>
                {line}
              </div>
            ))}
          </pre>
        </div>
      ))}
    </div>
  );
}
