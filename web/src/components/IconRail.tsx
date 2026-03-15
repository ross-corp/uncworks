import { useState, useRef, useEffect } from "react";
import { useTheme } from "../hooks/useTheme";

type FilterValue = "all" | "active" | "succeeded" | "failed";

interface IconRailProps {
  activeFilter: FilterValue;
  onFilterChange: (f: FilterValue) => void;
  onNewRun: () => void;
}

export function IconRail({ activeFilter, onFilterChange, onNewRun }: IconRailProps) {
  const [filterOpen, setFilterOpen] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);
  const { theme, toggleTheme } = useTheme();

  useEffect(() => {
    if (!filterOpen) return;
    function handleClick(e: MouseEvent) {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        setFilterOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [filterOpen]);

  const filterOptions: { value: FilterValue; label: string }[] = [
    { value: "all", label: "All" },
    { value: "active", label: "Active" },
    { value: "succeeded", label: "Done" },
    { value: "failed", label: "Failed" },
  ];

  const hasActiveFilter = activeFilter !== "all";

  return (
    <nav
      className="flex flex-col items-center gap-2 py-3 shrink-0"
      style={{
        width: "48px",
        borderRight: "1px solid var(--unc-border)",
        backgroundColor: "var(--unc-bg)",
      }}
    >
      <div className="relative" ref={popoverRef}>
        <button
          data-testid="icon-rail-filter"
          onClick={() => setFilterOpen(!filterOpen)}
          title="Filter runs"
          className="flex items-center justify-center"
          style={{
            width: "32px",
            height: "32px",
            borderRadius: "4px",
            color: hasActiveFilter ? "var(--unc-accent)" : "var(--unc-muted)",
            cursor: "pointer",
            backgroundColor: "transparent",
            border: "none",
          }}
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3" />
          </svg>
        </button>

        {filterOpen && (
          <div
            className="absolute left-full top-0 ml-2 z-50 py-1"
            style={{
              backgroundColor: "var(--unc-bg)",
              border: "1px solid var(--unc-border)",
              minWidth: "120px",
              boxShadow: "0 4px 12px rgba(0,0,0,0.15)",
            }}
          >
            {filterOptions.map((opt) => (
              <button
                key={opt.value}
                onClick={() => {
                  onFilterChange(opt.value);
                  setFilterOpen(false);
                }}
                className="block w-full text-left px-3 py-1.5 text-sm"
                style={{
                  backgroundColor: activeFilter === opt.value ? "var(--unc-border)" : "transparent",
                  color: "var(--unc-fg)",
                  cursor: "pointer",
                  border: "none",
                }}
              >
                {opt.label}
              </button>
            ))}
          </div>
        )}
      </div>

      <button
        data-testid="icon-rail-new-run"
        onClick={onNewRun}
        title="New run (n)"
        className="flex items-center justify-center"
        style={{
          width: "32px",
          height: "32px",
          borderRadius: "4px",
          color: "var(--unc-muted)",
          cursor: "pointer",
          backgroundColor: "transparent",
          border: "none",
        }}
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <line x1="12" y1="5" x2="12" y2="19" />
          <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
      </button>

      <button
        data-testid="icon-rail-theme"
        onClick={toggleTheme}
        title="Toggle theme"
        className="flex items-center justify-center"
        style={{
          width: "32px",
          height: "32px",
          borderRadius: "4px",
          color: "var(--unc-muted)",
          cursor: "pointer",
          backgroundColor: "transparent",
          border: "none",
        }}
      >
        {theme === "dark" ? (
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="12" cy="12" r="5" />
            <line x1="12" y1="1" x2="12" y2="3" />
            <line x1="12" y1="21" x2="12" y2="23" />
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
            <line x1="1" y1="12" x2="3" y2="12" />
            <line x1="21" y1="12" x2="23" y2="12" />
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
          </svg>
        ) : (
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
          </svg>
        )}
      </button>
    </nav>
  );
}
