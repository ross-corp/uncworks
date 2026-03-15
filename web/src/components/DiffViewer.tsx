import type { SpanDiff } from "../types/agent-run";

function classifyLine(line: string): string {
  if (line.startsWith("+") && !line.startsWith("+++")) return "text-secondary bg-secondary/10";
  if (line.startsWith("-") && !line.startsWith("---")) return "text-destructive bg-destructive/10";
  if (line.startsWith("@@")) return "text-primary";
  return "text-muted-foreground";
}

export default function DiffViewer({ diff }: { diff: SpanDiff }) {
  if (!diff.files || diff.files.length === 0) {
    return (
      <div className="flex items-center justify-center py-8 text-sm text-muted-foreground/60">
        No file changes in this span
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 overflow-y-auto p-2">
      {diff.files.map((file, i) => (
        <div key={i} className="border border-border overflow-hidden">
          {/* File header */}
          <div className="border-b border-border bg-muted px-3 py-1.5">
            <span className="font-mono text-xs text-muted-foreground">{file.path}</span>
          </div>

          {/* Diff content */}
          <pre className="overflow-x-auto p-3 text-xs leading-5 font-mono bg-background">
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
