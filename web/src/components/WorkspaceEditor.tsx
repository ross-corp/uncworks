import { useState, useEffect } from "react";
import type { Repository } from "../types/agent-run";
import type { Workspace } from "../hooks/useWorkspaces";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

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
        className="w-full max-w-lg border border-border bg-card shadow-2xl"
      >
        <div className="flex items-center justify-between border-b border-border px-5 py-3">
          <h2 className="text-sm font-semibold fx-glow">
            {workspace ? "Edit Workspace" : "New Workspace"}
          </h2>
          <Button type="button" variant="ghost" size="sm" onClick={onClose} aria-label="Close">
            &times;
          </Button>
        </div>

        <div className="flex max-h-[60vh] flex-col gap-4 overflow-y-auto p-5">
          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">
              Name
            </label>
            <Input
              data-testid="workspace-editor-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-workspace"
              autoFocus
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">
              Description
            </label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional description"
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">
              Repositories
            </label>
            <div className="space-y-2">
              {repos.map((repo, i) => (
                <div key={i} data-testid={`workspace-editor-repo-${i}`} className="flex items-center gap-2">
                  <div className="flex-1">
                    <Input
                      data-testid={`workspace-editor-repo-${i}-url`}
                      list="ws-known-repos"
                      value={repo.url}
                      onChange={(e) => updateRepo(i, "url", e.target.value)}
                      placeholder="https://github.com/org/repo"
                    />
                  </div>
                  <div className="w-24">
                    <Input
                      data-testid={`workspace-editor-repo-${i}-branch`}
                      value={repo.branch}
                      onChange={(e) => updateRepo(i, "branch", e.target.value)}
                      placeholder="main"
                    />
                  </div>
                  {repos.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => removeRepo(i)}
                      className="text-xs text-destructive"
                    >
                      &times;
                    </Button>
                  )}
                </div>
              ))}
            </div>
            <datalist id="ws-known-repos">
              {knownRepos.map((url) => (
                <option key={url} value={url} />
              ))}
            </datalist>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={addRepo}
              className="mt-2 text-xs"
            >
              + Add repo
            </Button>
          </div>
        </div>

        <div className="flex items-center justify-between border-t border-border px-5 py-3">
          <div>
            {workspace && onDelete && (
              confirmDelete ? (
                <div className="flex items-center gap-2">
                  <span className="text-xs text-destructive">Delete this workspace?</span>
                  <Button
                    data-testid="workspace-editor-delete-confirm"
                    type="button"
                    variant="destructive"
                    size="sm"
                    onClick={() => onDelete(workspace.id)}
                  >
                    Confirm
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => setConfirmDelete(false)}
                  >
                    Cancel
                  </Button>
                </div>
              ) : (
                <Button
                  data-testid="workspace-editor-delete"
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => setConfirmDelete(true)}
                  className="text-xs text-destructive"
                >
                  Delete
                </Button>
              )
            )}
          </div>
          <div className="flex gap-2">
            <Button type="button" variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button data-testid="workspace-editor-save" type="submit">
              {workspace ? "Save" : "Create"}
            </Button>
          </div>
        </div>
      </form>
    </div>
  );
}
