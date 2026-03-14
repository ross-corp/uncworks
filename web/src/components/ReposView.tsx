import { useState } from "react";

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
      <div className="flex items-center gap-2 border-b border-edge px-4 py-3">
        <input
          data-testid="repos-add-input"
          className="input-field flex-1"
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
        <button
          data-testid="repos-add-button"
          onClick={handleAdd}
          disabled={!newUrl.trim()}
          className="btn-primary text-sm disabled:opacity-40"
        >
          Add
        </button>
      </div>

      <table className="w-full text-sm" style={{ tableLayout: "fixed" }}>
        <colgroup>
          <col style={{ width: 200 }} />
          <col />
          <col style={{ width: 80 }} />
        </colgroup>
        <thead>
          <tr className="border-b border-edge text-left text-xs font-medium text-txt-tertiary">
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
              className="group border-b border-edge transition-colors hover:bg-surface-1"
            >
              <td className="px-4 py-2.5 font-medium text-txt-primary">
                {repoName(url)}
              </td>
              <td className="px-4 py-2.5 font-mono text-xs text-txt-secondary overflow-hidden text-ellipsis whitespace-nowrap">
                {url}
              </td>
              <td className="px-4 py-2.5 text-right">
                <button
                  data-testid={`repos-remove-${index}`}
                  onClick={() => onRemoveRepo(url)}
                  className="btn-ghost px-2 py-1 text-xs text-danger opacity-0 group-hover:opacity-100"
                >
                  Remove
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {repos.length === 0 && (
        <div className="px-6 py-12 text-center text-sm text-txt-tertiary">
          No repositories registered. Add one above or create an agent run.
        </div>
      )}
    </div>
  );
}
