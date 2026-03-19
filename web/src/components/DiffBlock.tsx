import { useState } from "react";

interface DiffBlockProps {
  content: string;
}

function parseHeader(lines: string[]): string | null {
  for (const line of lines) {
    if (line.startsWith("+++ b/")) return line.slice(6);
    if (line.startsWith("+++ ")) return line.slice(4);
  }
  return null;
}

function lineClass(line: string): string {
  if (line.startsWith("+")) return "bg-green-500/15 text-green-400";
  if (line.startsWith("-")) return "bg-red-500/15 text-red-400";
  return "text-muted-foreground";
}

export default function DiffBlock({ content }: DiffBlockProps) {
  const lines = content.split("\n");
  const filePath = parseHeader(lines);
  const [collapsed, setCollapsed] = useState(false);

  const diffLines = lines.filter(
    (l) => !l.startsWith("---") && !l.startsWith("+++") && !l.startsWith("diff --git"),
  );

  return (
    <div className="rounded border border-border bg-muted/40 overflow-hidden text-xs font-mono">
      {filePath && (
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-muted-foreground hover:text-foreground bg-muted/60"
        >
          <span>{collapsed ? "▶" : "▼"}</span>
          <span className="truncate">{filePath}</span>
        </button>
      )}
      {!collapsed && (
        <pre className="overflow-x-auto p-2 leading-5">
          {diffLines.map((line, i) => (
            <div key={i} className={`px-1 ${lineClass(line)}`}>
              {line || "\u00A0"}
            </div>
          ))}
        </pre>
      )}
    </div>
  );
}
