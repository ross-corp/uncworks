import { useState, useCallback, type KeyboardEvent } from "react";

interface CommandInputProps {
  onCommand: (cmd: string) => void;
  onFilter: (q: string) => void;
  placeholder?: string;
}

export default function CommandInput({ onCommand, onFilter, placeholder = "Type a message, :command, or /filter..." }: CommandInputProps) {
  const [value, setValue] = useState("");

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Escape") {
        setValue("");
        return;
      }
      if (e.key === "Enter" && value.trim()) {
        const trimmed = value.trim();
        if (trimmed.startsWith(":")) {
          onCommand(trimmed.slice(1));
        } else if (trimmed.startsWith("/")) {
          onFilter(trimmed.slice(1));
        } else {
          onCommand(trimmed);
        }
        setValue("");
      }
    },
    [value, onCommand, onFilter],
  );

  return (
    <div className="border-t border-border bg-background px-4 py-3">
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        className="w-full rounded-md border border-border bg-muted/40 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
      />
    </div>
  );
}
