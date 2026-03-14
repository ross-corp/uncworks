import { useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
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
  const writtenCountRef = useRef(0);
  const [userScrolledBack, setUserScrolledBack] = useState(false);

  // Initialize terminal
  useEffect(() => {
    if (!containerRef.current) return;

    const term = new Terminal({
      cursorBlink: false,
      disableStdin: true,
      convertEol: true,
      fontFamily: "'JetBrains Mono', monospace",
      fontSize: 13,
      theme: {
        background: "#1a1a2e",
        foreground: "#e0e0e0",
        cursor: "#e0e0e0",
      },
      scrollback: 10000,
    });

    const fit = new FitAddon();
    term.loadAddon(fit);
    term.loadAddon(new WebLinksAddon());

    term.open(containerRef.current);
    fit.fit();

    termRef.current = term;
    fitRef.current = fit;
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
      writtenCountRef.current = 0;
    };
  }, []);

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

  return (
    <div
      ref={containerRef}
      style={{ width: "100%", height: "100%" }}
    />
  );
}
