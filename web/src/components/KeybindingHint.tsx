// web/src/components/KeybindingHint.tsx — Display a keyboard shortcut hint
// with optional macOS symbol normalization.
import { isMac } from "../lib/wails-env";

/**
 * Normalize a key sequence string to human-readable display tokens.
 * On macOS: meta→⌘, ctrl→⌃, alt→⌥, shift→⇧
 * On other platforms: meta→Win, ctrl→Ctrl, alt→Alt, shift→Shift
 */
export function normalizeKeyDisplay(keys: string, mac: boolean): string {
  return keys
    .split(" ")
    .map((token) =>
      token
        .split("+")
        .map((part) => {
          if (mac) {
            switch (part.toLowerCase()) {
              case "meta":  return "⌘";
              case "ctrl":  return "⌃";
              case "alt":   return "⌥";
              case "shift": return "⇧";
              case "space": return "␣";
              case "escape": return "⎋";
              default: return part;
            }
          }
          switch (part.toLowerCase()) {
            case "meta":  return "Win";
            case "ctrl":  return "Ctrl";
            case "alt":   return "Alt";
            case "shift": return "Shift";
            case "space": return "Space";
            case "escape": return "Esc";
            default: return part;
          }
        })
        .join(mac ? "" : "+")
    )
    .join(" ");
}

interface KeybindingHintProps {
  /** Key sequence string e.g. "g r", "ctrl+s", "Escape" */
  keys: string;
  className?: string;
}

/**
 * Renders a keyboard shortcut hint as a series of <kbd> elements.
 * Automatically applies macOS symbol normalization on Apple platforms.
 */
export default function KeybindingHint({ keys, className }: KeybindingHintProps) {
  const mac = isMac();
  const display = normalizeKeyDisplay(keys, mac);
  const tokens = display.split(" ").filter(Boolean);

  return (
    <span className={`inline-flex items-center gap-0.5 ${className ?? ""}`}>
      {tokens.map((token, i) => (
        <kbd
          key={i}
          className="font-mono text-xs bg-muted border border-border px-1 py-0.5 rounded leading-none"
        >
          {token}
        </kbd>
      ))}
    </span>
  );
}
