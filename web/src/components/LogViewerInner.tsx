import { useEffect, useRef, useState, useCallback } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SearchAddon } from "@xterm/addon-search";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";

export default function LogViewerInner({
  lines,
  streaming,
}: {
  lines: string[];
  streaming: boolean;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const searchRef = useRef<SearchAddon | null>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const writtenCountRef = useRef(0);
  const [userScrolledBack, setUserScrolledBack] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchVisible, setSearchVisible] = useState(false);

  const focusSearch = useCallback(() => {
    setSearchVisible(true);
    // Delay focus to allow the input to render
    setTimeout(() => searchInputRef.current?.focus(), 0);
  }, []);

  // Initialize terminal
  useEffect(() => {
    if (!containerRef.current) return;

    const term = new Terminal({
      cursorBlink: false,
      disableStdin: true,
      convertEol: true,
      fontFamily: "'IoskeleyMono', monospace",
      fontSize: 13,
      theme: {
        background: "#000000",
        foreground: "#FFB000",
        cursor: "#FFB000",
      },
      scrollback: 10000,
    });

    const fit = new FitAddon();
    const search = new SearchAddon();
    term.loadAddon(fit);
    term.loadAddon(search);
    term.loadAddon(new WebLinksAddon());

    term.open(containerRef.current);
    fit.fit();

    termRef.current = term;
    fitRef.current = fit;
    searchRef.current = search;
    writtenCountRef.current = 0;

    const resizeObserver = new ResizeObserver(() => {
      fit.fit();
    });
    resizeObserver.observe(containerRef.current);

    // Detect user scrolling back
    term.onScroll(() => {
      if (!termRef.current) return;
      const buf = termRef.current.buffer.active;
      const isAtBottom =
        buf.viewportY >= buf.baseY;
      setUserScrolledBack(!isAtBottom);
    });

    return () => {
      resizeObserver.disconnect();
      term.dispose();
      termRef.current = null;
      fitRef.current = null;
      searchRef.current = null;
      writtenCountRef.current = 0;
    };
  }, []);

  // Global Ctrl+F handler
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "f") {
        e.preventDefault();
        focusSearch();
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [focusSearch]);

  // Write lines to terminal
  useEffect(() => {
    const term = termRef.current;
    if (!term) return;

    const alreadyWritten = writtenCountRef.current;

    if (lines.length < alreadyWritten) {
      // Lines were reset (e.g. new run) — clear terminal
      term.clear();
      writtenCountRef.current = 0;
    }

    const start = writtenCountRef.current;
    for (let i = start; i < lines.length; i++) {
      term.writeln(lines[i]);
    }
    writtenCountRef.current = lines.length;

    // Auto-scroll to bottom in streaming mode unless user scrolled back
    if (streaming && !userScrolledBack) {
      term.scrollToBottom();
    }
  }, [lines, streaming, userScrolledBack]);

  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    if (searchRef.current && value) {
      searchRef.current.findNext(value);
    }
  };

  const handleSearchKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      if (e.shiftKey) {
        searchRef.current?.findPrevious(searchQuery);
      } else {
        searchRef.current?.findNext(searchQuery);
      }
    }
    if (e.key === "Escape") {
      setSearchVisible(false);
      searchRef.current?.clearDecorations();
    }
  };

  const handleCopyAll = async () => {
    const allText = lines.join("\n");
    try {
      await navigator.clipboard.writeText(allText);
    } catch {
      // Fallback: create a temporary textarea
      const ta = document.createElement("textarea");
      ta.value = allText;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      document.body.removeChild(ta);
    }
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", width: "100%", height: "100%" }}>
      {/* Toolbar */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "8px",
          padding: "4px 8px",
          background: "#111",
          borderBottom: "1px solid #333",
          flexShrink: 0,
        }}
      >
        {searchVisible ? (
          <input
            ref={searchInputRef}
            type="text"
            placeholder="Search logs..."
            value={searchQuery}
            onChange={(e) => handleSearchChange(e.target.value)}
            onKeyDown={handleSearchKeyDown}
            style={{
              flex: 1,
              maxWidth: "300px",
              padding: "2px 8px",
              fontSize: "12px",
              background: "#222",
              color: "#FFB000",
              border: "1px solid #444",
              borderRadius: "3px",
              outline: "none",
              fontFamily: "'IoskeleyMono', monospace",
            }}
          />
        ) : (
          <button
            onClick={focusSearch}
            title="Search (Ctrl+F)"
            style={{
              padding: "2px 8px",
              fontSize: "12px",
              background: "#222",
              color: "#999",
              border: "1px solid #444",
              borderRadius: "3px",
              cursor: "pointer",
              fontFamily: "'IoskeleyMono', monospace",
            }}
          >
            Search
          </button>
        )}
        <button
          onClick={handleCopyAll}
          title="Copy all log content"
          style={{
            padding: "2px 8px",
            fontSize: "12px",
            background: "#222",
            color: "#999",
            border: "1px solid #444",
            borderRadius: "3px",
            cursor: "pointer",
            fontFamily: "'IoskeleyMono', monospace",
          }}
        >
          Copy all
        </button>
      </div>
      {/* Terminal */}
      <div
        ref={containerRef}
        style={{ width: "100%", flex: 1, minHeight: 0 }}
      />
    </div>
  );
}
