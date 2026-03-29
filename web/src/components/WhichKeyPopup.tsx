// web/src/components/WhichKeyPopup.tsx — Floating which-key popup that shows
// available chord completions after a prefix key is pressed.
import { useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import type { WhichKeyEntry } from "../contexts/KeybindingsContext";

interface WhichKeyPopupProps {
  visible: boolean;
  prefix: string | null;
  entries: WhichKeyEntry[];
  onDismiss: () => void;
}

/**
 * Fixed bottom-right popup with a monospace key column and a description column.
 * Uses CSS transitions for fade + translateY animation.
 * A pointerdown outside the popup calls onDismiss.
 */
export default function WhichKeyPopup({
  visible,
  prefix,
  entries,
  onDismiss,
}: WhichKeyPopupProps) {
  const popupRef = useRef<HTMLDivElement>(null);

  // Dismiss on pointerdown outside the popup.
  useEffect(() => {
    if (!visible) return;
    function onPointerDown(e: PointerEvent) {
      if (popupRef.current && !popupRef.current.contains(e.target as Node)) {
        onDismiss();
      }
    }
    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, [visible, onDismiss]);

  const popup = (
    <div
      ref={popupRef}
      role="dialog"
      aria-label={`Which-key: completions for ${prefix ?? ""}`}
      aria-live="polite"
      style={{
        position: "fixed",
        bottom: 24,
        right: 24,
        zIndex: 9999,
        transition: "opacity 120ms ease, transform 120ms ease",
        opacity: visible ? 1 : 0,
        transform: visible ? "translateY(0)" : "translateY(8px)",
        pointerEvents: visible ? "auto" : "none",
      }}
      className="rounded-lg border bg-popover text-popover-foreground shadow-lg min-w-[200px] max-w-[320px]"
    >
      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-2 border-b">
        <span className="text-xs text-muted-foreground">
          {prefix ? (
            <>
              <kbd className="font-mono text-xs bg-muted px-1 py-0.5 rounded">{prefix}</kbd>
              {" "}→
            </>
          ) : "Keybindings"}
        </span>
      </div>

      {/* Entries */}
      {entries.length === 0 ? (
        <div className="px-3 py-2 text-xs text-muted-foreground">No completions</div>
      ) : (
        <div className="divide-y divide-border/50">
          {entries.map((entry) => (
            <div key={entry.action} className="flex items-center gap-3 px-3 py-1.5">
              <kbd className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded shrink-0 min-w-[2ch] text-center">
                {entry.key}
              </kbd>
              <span className="text-xs text-foreground truncate">{entry.description}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );

  return createPortal(popup, document.body);
}
