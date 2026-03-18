import { useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { apiWsUrl } from "../hooks/apiFetch";
import "@xterm/xterm/css/xterm.css";

type ConnectionStatus = "connecting" | "connected" | "disconnected";

export default function ShellTerminalInner({ runId }: { runId: string }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [status, setStatus] = useState<ConnectionStatus>("connecting");

  useEffect(() => {
    if (!containerRef.current) return;

    const term = new Terminal({
      cursorBlink: true,
      convertEol: true,
      fontFamily: "'IoskeleyMono', monospace",
      fontSize: 13,
      theme: {
        background: "#000000",
        foreground: "#FFB000",
        cursor: "#FFB000",
      },
      scrollback: 5000,
    });

    const fit = new FitAddon();
    term.loadAddon(fit);
    term.loadAddon(new WebLinksAddon());

    term.open(containerRef.current);
    fit.fit();

    termRef.current = term;
    fitRef.current = fit;

    // Connect WebSocket
    const wsUrl = apiWsUrl(`/api/v1/runs/${runId}/exec`);
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setStatus("connected");
      // Send initial resize
      const dims = { type: "resize", cols: term.cols, rows: term.rows };
      ws.send(JSON.stringify(dims));
    };

    ws.onmessage = (event) => {
      term.write(event.data);
    };

    ws.onerror = () => {
      setStatus("disconnected");
    };

    ws.onclose = () => {
      setStatus("disconnected");
      term.write("\r\n\x1b[33m[Disconnected]\x1b[0m\r\n");
    };

    // Send keystrokes to WebSocket
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });

    // Handle resize
    const resizeObserver = new ResizeObserver(() => {
      fit.fit();
      if (ws.readyState === WebSocket.OPEN) {
        const dims = { type: "resize", cols: term.cols, rows: term.rows };
        ws.send(JSON.stringify(dims));
      }
    });
    resizeObserver.observe(containerRef.current);

    return () => {
      resizeObserver.disconnect();
      ws.close();
      term.dispose();
      termRef.current = null;
      fitRef.current = null;
      wsRef.current = null;
    };
  }, [runId]);

  const statusLabel: Record<ConnectionStatus, string> = {
    connecting: "Connecting...",
    connected: "Connected",
    disconnected: "Disconnected",
  };

  const statusColor: Record<ConnectionStatus, string> = {
    connecting: "text-primary",
    connected: "text-secondary",
    disconnected: "text-destructive",
  };

  return (
    <div className="flex h-full flex-col">
      {/* Status bar */}
      <div className="flex items-center justify-between border-b border-border bg-muted px-3 py-1">
        <span className="text-xs text-muted-foreground/60">Shell</span>
        <span className={`text-xs ${statusColor[status]}`}>
          {statusLabel[status]}
        </span>
      </div>

      {/* Terminal container */}
      <div ref={containerRef} className="flex-1" />
    </div>
  );
}
