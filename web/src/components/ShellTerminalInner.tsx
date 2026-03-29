import { useEffect, useRef, useState, useCallback } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { apiFetch, apiWsUrl } from "../hooks/apiFetch";
import { useThemeNew } from "../hooks/useThemeNew";
import type { AgentRunPhase } from "../types/agent-run";
import "@xterm/xterm/css/xterm.css";

const DARK_TERMINAL_THEME = {
  background: "#0a0a0a",
  foreground: "#e5e5e5",
  cursor: "#e5e5e5",
  selectionBackground: "#264f78",
  black: "#000000",
  red: "#cd3131",
  green: "#0dbc79",
  yellow: "#e5e510",
  blue: "#2472c8",
  magenta: "#bc3fbc",
  cyan: "#11a8cd",
  white: "#e5e5e5",
};

const LIGHT_TERMINAL_THEME = {
  background: "#ffffff",
  foreground: "#1e1e1e",
  cursor: "#1e1e1e",
  selectionBackground: "#add6ff",
  black: "#000000",
  red: "#cd3131",
  green: "#008000",
  yellow: "#795e26",
  blue: "#0451a5",
  magenta: "#bc05bc",
  cyan: "#0598bc",
  white: "#1e1e1e",
};

type ConnectionStatus =
  | "connecting"
  | "connected"
  | "disconnected"
  | "pod_down"
  | "starting_debug";

export default function ShellTerminalInner({
  runId,
  phase,
}: {
  runId: string;
  phase: AgentRunPhase;
}) {
  const { resolvedTheme } = useThemeNew();
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const dataDisposableRef = useRef<{ dispose: () => void } | null>(null);
  const [status, setStatus] = useState<ConnectionStatus>("connecting");
  const everConnectedRef = useRef(false);

  const podIsDown =
    phase === "succeeded" || phase === "failed" || phase === "cancelled";

  const connectWs = useCallback(() => {
    const term = termRef.current;
    if (!term) return;

    // Close existing connection
    if (wsRef.current) {
      wsRef.current.onclose = null;
      wsRef.current.onerror = null;
      wsRef.current.close();
    }

    // Clean up old data handler to prevent stacking
    if (dataDisposableRef.current) {
      dataDisposableRef.current.dispose();
      dataDisposableRef.current = null;
    }

    setStatus("connecting");
    const wsUrl = apiWsUrl(`/api/v1/runs/${runId}/exec`);
    const ws = new WebSocket(wsUrl);
    ws.binaryType = "arraybuffer"; // Receive binary as ArrayBuffer, not Blob
    wsRef.current = ws;

    ws.onopen = () => {
      everConnectedRef.current = true;
      setStatus("connected");
      // Send initial resize
      const dims = { type: "resize", cols: term.cols, rows: term.rows };
      ws.send(JSON.stringify(dims));
    };

    ws.onmessage = (event) => {
      // Backend sends BinaryMessage — decode ArrayBuffer to string for xterm
      if (event.data instanceof ArrayBuffer) {
        const text = new TextDecoder().decode(event.data);
        term.write(text);
      } else {
        term.write(event.data);
      }
    };

    ws.onerror = () => {
      if (!everConnectedRef.current) {
        setStatus("pod_down");
      }
    };

    ws.onclose = () => {
      if (!everConnectedRef.current) {
        setStatus("pod_down");
      } else {
        setStatus("disconnected");
        term.write("\r\n\x1b[33m[Disconnected — press Reconnect to resume]\x1b[0m\r\n");
      }
    };

    // Register keystroke handler (dispose old one first)
    dataDisposableRef.current = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });
  }, [runId]);

  // Initialize terminal + connect
  useEffect(() => {
    if (!containerRef.current) return;

    const termTheme = resolvedTheme === "dark" ? DARK_TERMINAL_THEME : LIGHT_TERMINAL_THEME;
    const term = new Terminal({
      cursorBlink: true,
      convertEol: true,
      fontFamily: "'IoskeleyMono', monospace",
      fontSize: 13,
      theme: termTheme,
      scrollback: 5000,
    });

    const fit = new FitAddon();
    term.loadAddon(fit);
    term.loadAddon(new WebLinksAddon());

    term.open(containerRef.current);
    fit.fit();

    termRef.current = term;
    fitRef.current = fit;

    // Only auto-connect if pod should be running
    if (!podIsDown) {
      connectWs();
    } else {
      setStatus("pod_down");
    }

    const resizeObserver = new ResizeObserver(() => {
      fit.fit();
      const ws = wsRef.current;
      if (ws && ws.readyState === WebSocket.OPEN) {
        const dims = { type: "resize", cols: term.cols, rows: term.rows };
        ws.send(JSON.stringify(dims));
      }
    });
    resizeObserver.observe(containerRef.current);

    return () => {
      resizeObserver.disconnect();
      if (dataDisposableRef.current) {
        dataDisposableRef.current.dispose();
        dataDisposableRef.current = null;
      }
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.close();
      }
      term.dispose();
      termRef.current = null;
      fitRef.current = null;
      wsRef.current = null;
    };
  }, [runId]); // eslint-disable-line react-hooks/exhaustive-deps

  // Update terminal theme when site theme changes
  useEffect(() => {
    if (termRef.current) {
      termRef.current.options.theme = resolvedTheme === "dark" ? DARK_TERMINAL_THEME : LIGHT_TERMINAL_THEME;
    }
  }, [resolvedTheme]);

  const reconnect = useCallback(() => {
    termRef.current?.write("\r\n\x1b[36m[Reconnecting...]\x1b[0m\r\n");
    everConnectedRef.current = false;
    connectWs();
  }, [connectWs]);

  const startDebug = useCallback(async () => {
    try {
      setStatus("starting_debug");
      termRef.current?.write("\r\n\x1b[36m[Starting debug session...]\x1b[0m\r\n");
      const res = await apiFetch(`/api/v1/runs/${runId}/debug`, { method: "POST" });
      if (!res.ok) {
        const msg = await res.text().catch(() => "unknown error");
        termRef.current?.write(`\r\n\x1b[31m[Failed: ${msg}]\x1b[0m\r\n`);
        setStatus("pod_down");
        return;
      }
      termRef.current?.write("\r\n\x1b[36m[Pod starting, connecting in 5s...]\x1b[0m\r\n");
      setTimeout(() => {
        everConnectedRef.current = false;
        connectWs();
      }, 5000);
    } catch {
      termRef.current?.write("\r\n\x1b[31m[Failed to start debug session]\x1b[0m\r\n");
      setStatus("pod_down");
    }
  }, [runId, connectWs]);

  const stopDebug = useCallback(async () => {
    try {
      await apiFetch(`/api/v1/runs/${runId}/debug`, { method: "DELETE" });
      termRef.current?.write("\r\n\x1b[33m[Debug session stopped]\x1b[0m\r\n");
      if (wsRef.current) wsRef.current.close();
      setStatus("pod_down");
    } catch {
      termRef.current?.write("\r\n\x1b[31m[Failed to stop debug session]\x1b[0m\r\n");
    }
  }, [runId]);

  return (
    <div className="flex h-full flex-col">
      {/* Status bar */}
      <div className="flex items-center justify-between border-b border-border bg-muted px-3 py-1">
        <span className="text-xs text-muted-foreground/60">
          Shell &mdash; {runId}
        </span>
        <div className="flex items-center gap-2">
          <span className={`text-xs ${
            status === "connected" ? "text-emerald-500" :
            status === "connecting" || status === "starting_debug" ? "text-amber-500" :
            "text-destructive"
          }`}>
            {status === "connecting" && "Connecting..."}
            {status === "connected" && "Connected"}
            {status === "disconnected" && "Disconnected"}
            {status === "pod_down" && (podIsDown ? `Pod stopped (${phase})` : "Pod not running")}
            {status === "starting_debug" && "Starting pod..."}
          </span>
          {status === "disconnected" && (
            <button
              onClick={reconnect}
              aria-label="Reconnect terminal"
              className="px-2 py-0.5 text-xs bg-primary text-primary-foreground hover:opacity-90"
            >
              Reconnect
            </button>
          )}
          {status === "connected" && (
            <button
              onClick={stopDebug}
              aria-label="Stop debug session"
              className="px-2 py-0.5 text-xs bg-destructive text-destructive-foreground hover:opacity-90"
            >
              Stop
            </button>
          )}
        </div>
      </div>

      {/* Terminal container */}
      <div ref={containerRef} className="flex-1 relative">
        {/* Overlay for pod_down state */}
        {status === "pod_down" && (
          <div className="absolute inset-0 z-10 flex flex-col items-center justify-center gap-3 bg-background/90">
            <p className="text-sm text-amber-500">
              {podIsDown
                ? `Run ${phase}. Start a debug session to explore the workspace.`
                : "Pod is not running."}
            </p>
            <button
              onClick={startDebug}
              aria-label="Start debug session"
              className="px-3 py-1.5 text-sm bg-primary text-primary-foreground hover:opacity-90"
            >
              Start Debug Session
            </button>
          </div>
        )}
        {status === "starting_debug" && (
          <div className="absolute inset-0 z-10 flex flex-col items-center justify-center gap-3 bg-background/90">
            <p className="text-sm text-primary animate-pulse">Starting pod...</p>
          </div>
        )}
      </div>
    </div>
  );
}
