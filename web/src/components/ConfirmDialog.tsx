import { useEffect } from "react";

export default function ConfirmDialog({
  title,
  message,
  confirmLabel = "Delete",
  onConfirm,
  onCancel,
}: {
  title: string;
  message: string;
  confirmLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onCancel();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onCancel]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/60 pt-[10vh]"
      onClick={(e) => {
        if (e.target === e.currentTarget) onCancel();
      }}
    >
      <div className="w-full max-w-sm rounded-lg border border-edge bg-surface-1 shadow-2xl">
        <div className="border-b border-edge px-5 py-3">
          <h2 className="text-sm font-semibold">{title}</h2>
        </div>
        <div className="px-5 py-4 text-sm text-txt-secondary">{message}</div>
        <div className="flex justify-end gap-2 border-t border-edge px-5 py-3">
          <button onClick={onCancel} className="btn-ghost">
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className="rounded bg-danger px-3 py-1.5 text-sm font-medium text-white hover:bg-danger/80 transition-colors"
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
