import { useState, useEffect, useCallback, useRef, type ReactNode } from "react";

const STORAGE_KEY = "clean-ui-pane-width";

function loadWidth(): number | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const px = parseInt(raw, 10);
      if (!isNaN(px) && px >= 200 && px <= window.innerWidth * 0.8) {
        return px;
      }
    }
  } catch { /* ignore */ }
  return null;
}

function saveWidth(px: number) {
  localStorage.setItem(STORAGE_KEY, String(px));
}

interface SplitPaneProps {
  detailOpen: boolean;
  children: [ReactNode, ReactNode];
}

export default function SplitPane({ detailOpen, children }: SplitPaneProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [detailWidth, setDetailWidth] = useState<number>(() => loadWidth() ?? 0);
  const dragging = useRef(false);

  // When detail opens, set default width if none stored
  useEffect(() => {
    if (detailOpen && detailWidth === 0) {
      const half = Math.floor((containerRef.current?.clientWidth ?? window.innerWidth) / 2);
      setDetailWidth(half);
    }
  }, [detailOpen, detailWidth]);

  const onMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    dragging.current = true;

    const onMouseMove = (ev: MouseEvent) => {
      if (!dragging.current || !containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const rightWidth = rect.right - ev.clientX;
      const clamped = Math.max(200, Math.min(rightWidth, rect.width * 0.8));
      setDetailWidth(clamped);
    };

    const onMouseUp = () => {
      dragging.current = false;
      // Save on release
      setDetailWidth((w) => {
        saveWidth(w);
        return w;
      });
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };

    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
  }, []);

  const gridColumns = detailOpen
    ? `1fr ${detailWidth}px`
    : "1fr 0px";

  return (
    <div
      ref={containerRef}
      className="flex-1 overflow-hidden"
      style={{
        display: "grid",
        gridTemplateColumns: gridColumns,
        transition: dragging.current ? "none" : "grid-template-columns 200ms ease",
      }}
    >
      {/* Left pane */}
      <div className="overflow-hidden min-w-0">{children[0]}</div>

      {/* Drag handle + right pane */}
      {detailOpen && (
        <>
          <div style={{ display: "contents" }}>
            <div
              onMouseDown={onMouseDown}
              className="absolute z-10"
              style={{
                width: "4px",
                cursor: "col-resize",
                height: "100%",
                marginLeft: "-2px",
                backgroundColor: "transparent",
              }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLElement).style.backgroundColor = "var(--color-border)";
              }}
              onMouseLeave={(e) => {
                if (!dragging.current) {
                  (e.currentTarget as HTMLElement).style.backgroundColor = "transparent";
                }
              }}
            />
          </div>
        </>
      )}

      {/* Right pane */}
      <div
        className="overflow-hidden min-w-0"
        style={{
          borderLeft: detailOpen ? "1px solid var(--color-border)" : "none",
        }}
      >
        {children[1]}
      </div>
    </div>
  );
}
