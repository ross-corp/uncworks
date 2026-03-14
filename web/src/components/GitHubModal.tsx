import { useState, useEffect } from "react";

type GitHubModalProps = {
  mode: "load" | "push";
  onLoad?: (repo: string, path: string) => void;
  onPush?: (repo: string, path: string, message: string) => void;
  onClose: () => void;
};

export default function GitHubModal({
  mode,
  onLoad,
  onPush,
  onClose,
}: GitHubModalProps) {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  const [repo, setRepo] = useState("");
  const [path, setPath] = useState("");
  const [message, setMessage] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!repo.trim() || !path.trim()) return;
    if (mode === "load") {
      onLoad?.(repo.trim(), path.trim());
    } else {
      if (!message.trim()) return;
      onPush?.(repo.trim(), path.trim(), message.trim());
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/60 pt-[10vh]"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <form
        data-testid="github-modal"
        onSubmit={handleSubmit}
        className="w-full max-w-sm rounded-lg border border-edge bg-surface-1 shadow-2xl"
      >
        <div className="flex items-center justify-between border-b border-edge px-5 py-3">
          <h2 className="text-sm font-semibold">
            {mode === "load" ? "Load Spec from GitHub" : "Push Spec to GitHub"}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="btn-ghost px-2"
            aria-label="Close"
          >
            &times;
          </button>
        </div>

        <div className="flex flex-col gap-4 p-5">
          <div>
            <label className="mb-1 block text-xs font-medium text-txt-secondary">
              Repository
            </label>
            <input
              data-testid="github-modal-repo"
              className="input-field"
              value={repo}
              onChange={(e) => setRepo(e.target.value)}
              placeholder="owner/repo"
              autoFocus
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-txt-secondary">
              File Path
            </label>
            <input
              data-testid="github-modal-path"
              className="input-field"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              placeholder="specs/my-spec.md"
            />
          </div>

          {mode === "push" && (
            <div>
              <label className="mb-1 block text-xs font-medium text-txt-secondary">
                Commit Message
              </label>
              <input
                className="input-field"
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                placeholder="Update spec"
              />
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 border-t border-edge px-5 py-3">
          <button type="button" onClick={onClose} className="btn-ghost">
            Cancel
          </button>
          <button data-testid="github-modal-submit" type="submit" className="btn-primary">
            {mode === "load" ? "Load" : "Push"}
          </button>
        </div>
      </form>
    </div>
  );
}
