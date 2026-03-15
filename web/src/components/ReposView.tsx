import { useState } from "react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

function repoName(url: string): string {
  const parts = url.replace(/\.git$/, "").split("/");
  return parts[parts.length - 1] || url;
}

export default function ReposView({
  repos,
  onAddRepo,
  onRemoveRepo,
}: {
  repos: string[];
  onAddRepo: (url: string) => void;
  onRemoveRepo: (url: string) => void;
}) {
  const [newUrl, setNewUrl] = useState("");

  function handleAdd() {
    const url = newUrl.trim();
    if (!url) return;
    onAddRepo(url);
    setNewUrl("");
  }

  return (
    <div className="overflow-x-auto">
      {/* Add Repository */}
      <div className="flex items-center gap-2 border-b border-border px-4 py-3">
        <Input
          data-testid="repos-add-input"
          className="flex-1"
          value={newUrl}
          onChange={(e) => setNewUrl(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              handleAdd();
            }
          }}
          placeholder="https://github.com/org/repo"
        />
        <Button
          data-testid="repos-add-button"
          onClick={handleAdd}
          disabled={!newUrl.trim()}
          className="text-sm disabled:opacity-40"
        >
          Add
        </Button>
      </div>

      <table className="w-full text-sm" style={{ tableLayout: "fixed" }}>
        <colgroup>
          <col style={{ width: 200 }} />
          <col />
          <col style={{ width: 80 }} />
        </colgroup>
        <thead>
          <tr className="border-b border-border text-left text-xs font-medium text-muted-foreground/60">
            <th className="px-4 py-2">Name</th>
            <th className="px-4 py-2">URL</th>
            <th className="px-4 py-2" />
          </tr>
        </thead>
        <tbody>
          {repos.map((url, index) => (
            <tr
              key={url}
              data-testid={`repos-row-${index}`}
              className="group border-b border-border transition-colors hover:bg-card"
            >
              <td className="px-4 py-2.5 font-medium text-foreground">
                {repoName(url)}
              </td>
              <td className="px-4 py-2.5 font-mono text-xs text-muted-foreground overflow-hidden text-ellipsis whitespace-nowrap">
                {url}
              </td>
              <td className="px-4 py-2.5 text-right">
                <Button
                  data-testid={`repos-remove-${index}`}
                  variant="ghost"
                  size="sm"
                  onClick={() => onRemoveRepo(url)}
                  className="text-xs text-destructive opacity-0 group-hover:opacity-100"
                >
                  Remove
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {repos.length === 0 && (
        <div className="px-6 py-12 text-center text-sm text-muted-foreground/60">
          No repositories registered. Add one above or create an agent run.
        </div>
      )}
    </div>
  );
}
