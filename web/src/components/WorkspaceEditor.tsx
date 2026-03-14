import { useState, useEffect } from "react";
import type { Repository } from "../types/agent-run";
import type { Workspace } from "../hooks/useWorkspaces";

export default function WorkspaceEditor({
  workspace,
  knownRepos,
  onSave,
  onDelete,
  onClose,
}: {
  workspace?: Workspace;
  knownRepos: string[];
  onSave: (data: { name: string; description: string; repos: Repository[] }) => void;
  onDelete?: (id: string) => void;
  onClose: () => void;
}) {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  const [name, setName] = useState(workspace?.name ?? "");
  const [description, setDescription] = useState(workspace?.description ?? "");
  const [repos, setRepos] = useState<Repository[]>(
    workspace?.repos?.length ? workspace.repos.map((r) => ({ ...r })) : [{ url: "", branch: "main" }]
  );
  const [confirmDelete, setConfirmDelete] = useState(false);

  function updateRepo(index: number, field: keyof Repository, value: string) {
    setRepos((prev) => prev.map((r, i) => (i === index ? { ...r, [field]: value } : r)));
  }

  function removeRepo(index: number) {
    setRepos((prev) => (prev.length <= 1 ? prev : prev.filter((_, i) => i !== index)));
  }

  function addRepo() {
    setRepos((prev) => [...prev, { url: "", branch: "main" }]);
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    const validRepos = repos.filter((r) => r.url.trim());
    onSave({
      name: name.trim(),
      description: description.trim(),
      repos: validRepos.map((r) => ({ url: r.url.trim(), branch: r.branch.trim() || "main" })),
    });
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/60 pt-[10vh]"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <form
        data-testid="workspace-editor"
        onSubmit={handleSubmit}
        className="w-full max-w-lg rounded-lg border border-edge bg-surface-1 shadow-2xl"
      >
        <div className="flex items-center justify-between border-b border-edge px-5 py-3">
          <h2 className="text-sm font-semibold">
            {workspace ? "Edit Workspace" : "New Workspace"}
          </h2>
          <button type="button" onClick={onClose} className="btn-ghost px-2" aria-label="Close">
            &times;
          </button>
        </div>

        <div className="flex max-h-[60vh] flex-col gap-4 overflow-y-auto p-5">
          <div>
            <label className="mb-1 block text-xs font-medium text-txt-secondary">
              Name
            </label>
            <input
              data-testid="workspace-editor-name"
              className="input-field"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-workspace"
              autoFocus
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-txt-secondary">
              Description
            </label>
            <input
              className="input-field"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional description"
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-txt-secondary">
              Repositories
            </label>
            <div className="space-y-2">
              {repos.map((repo, i) => (
                <div key={i} data-testid={`workspace-editor-repo-${i}`} className="flex items-center gap-2">
                  <div className="flex-1">
                    <input
                      data-testid={`workspace-editor-repo-${i}-url`}
                      list="ws-known-repos"
                      className="input-field"
                      value={repo.url}
                      onChange={(e) => updateRepo(i, "url", e.target.value)}
                      placeholder="https://github.com/org/repo"
                    />
                  </div>
                  <div className="w-24">
                    <input
                      data-testid={`workspace-editor-repo-${i}-branch`}
                      className="input-field"
                      value={repo.branch}
                      onChange={(e) => updateRepo(i, "branch", e.target.value)}
                      placeholder="main"
                    />
                  </div>
                  {repos.length > 1 && (
                    <button
                      type="button"
                      onClick={() => removeRepo(i)}
                      className="btn-ghost px-2 text-xs text-danger"
                    >
                      &times;
                    </button>
                  )}
                </div>
              ))}
            </div>
            <datalist id="ws-known-repos">
              {knownRepos.map((url) => (
                <option key={url} value={url} />
              ))}
            </datalist>
            <button
              type="button"
              onClick={addRepo}
              className="btn-ghost mt-2 text-xs"
            >
              + Add repo
            </button>
          </div>
        </div>

        <div className="flex items-center justify-between border-t border-edge px-5 py-3">
          <div>
            {workspace && onDelete && (
              confirmDelete ? (
                <div className="flex items-center gap-2">
                  <span className="text-xs text-danger">Delete this workspace?</span>
                  <button
                    data-testid="workspace-editor-delete-confirm"
                    type="button"
                    onClick={() => onDelete(workspace.id)}
                    className="rounded bg-danger px-2 py-1 text-xs font-medium text-white hover:bg-danger/80 transition-colors"
                  >
                    Confirm
                  </button>
                  <button
                    type="button"
                    onClick={() => setConfirmDelete(false)}
                    className="btn-ghost text-xs"
                  >
                    Cancel
                  </button>
                </div>
              ) : (
                <button
                  data-testid="workspace-editor-delete"
                  type="button"
                  onClick={() => setConfirmDelete(true)}
                  className="btn-ghost text-xs text-danger"
                >
                  Delete
                </button>
              )
            )}
          </div>
          <div className="flex gap-2">
            <button type="button" onClick={onClose} className="btn-ghost">
              Cancel
            </button>
            <button data-testid="workspace-editor-save" type="submit" className="btn-primary">
              {workspace ? "Save" : "Create"}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
