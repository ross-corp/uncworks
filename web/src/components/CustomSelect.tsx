// web/src/components/CustomSelect.tsx
// Custom dropdown replacing native <select> for WKWebView (Wails/macOS) compatibility.
// Uses a details/summary pattern — no external dependencies.

import { useEffect, useRef } from "react";
import { cn } from "../lib/utils";

export interface SelectOption {
  value: string;
  label: string;
}

interface CustomSelectProps {
  value: string;
  onChange: (value: string) => void;
  options: SelectOption[];
  placeholder?: string;
  className?: string;
  disabled?: boolean;
}

export default function CustomSelect({
  value,
  onChange,
  options,
  placeholder = "Select…",
  className,
  disabled = false,
}: CustomSelectProps) {
  const detailsRef = useRef<HTMLDetailsElement>(null);

  const selectedLabel =
    options.find((o) => o.value === value)?.label ?? placeholder;

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (detailsRef.current && !detailsRef.current.contains(e.target as Node)) {
        detailsRef.current.open = false;
      }
    }
    document.addEventListener("click", handleClick);
    return () => document.removeEventListener("click", handleClick);
  }, []);

  function handleSelect(optValue: string) {
    onChange(optValue);
    if (detailsRef.current) detailsRef.current.open = false;
  }

  return (
    <details
      ref={detailsRef}
      className={cn(
        "relative",
        disabled && "opacity-50 pointer-events-none",
        className
      )}
    >
      <summary
        className={cn(
          "flex h-9 w-full items-center justify-between gap-2 rounded-md border border-input bg-background px-3 py-1 text-sm cursor-pointer list-none select-none",
          "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
          !value && "text-muted-foreground"
        )}
      >
        <span className="truncate">{selectedLabel}</span>
        <svg
          className="size-4 shrink-0 opacity-50"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </summary>
      <ul
        className={cn(
          "absolute z-50 mt-1 w-full min-w-[8rem] overflow-auto rounded-md border bg-popover text-popover-foreground shadow-md",
          "max-h-60 p-1"
        )}
      >
        {options.map((opt) => (
          <li
            key={opt.value}
            onClick={() => handleSelect(opt.value)}
            className={cn(
              "relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none",
              "hover:bg-accent hover:text-accent-foreground",
              opt.value === value && "bg-accent text-accent-foreground"
            )}
          >
            {opt.label}
          </li>
        ))}
      </ul>
    </details>
  );
}
