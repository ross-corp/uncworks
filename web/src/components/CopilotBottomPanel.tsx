import { useEffect, useRef, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";
import { useCopilotContextValue } from "../hooks/useCopilotContext";
import { useChatStreamSession } from "../hooks/useChatStreamSession";

// ── Guidance action parsing ─────────────────────────────────────────────────

function parseGuidanceActions(text: string): {
  cleaned: string;
  navPath: string | null;
  highlights: string[];
} {
  let cleaned = text;
  let navPath: string | null = null;
  const highlights: string[] = [];

  // [NAV:/path]
  cleaned = cleaned.replace(/\[NAV:([^\]]+)\]/g, (_, path) => {
    navPath = path.trim();
    return "";
  });

  // [HIGHLIGHT:selector]
  cleaned = cleaned.replace(/\[HIGHLIGHT:([^\]]+)\]/g, (_, sel) => {
    highlights.push(sel.trim());
    return "";
  });

  return { cleaned: cleaned.trim(), navPath, highlights };
}

function applyHighlight(selector: string) {
  try {
    const els = document.querySelectorAll<HTMLElement>(selector);
    els.forEach((el) => {
      el.classList.add("ring-2", "ring-primary", "ring-offset-1");
      setTimeout(() => {
        el.classList.remove("ring-2", "ring-primary", "ring-offset-1");
      }, 3000);
    });
  } catch {
    // Invalid selector — ignore
  }
}

// ── Relative time helper ────────────────────────────────────────────────────

function relativeTime(ts: number): string {
  const diff = Date.now() - ts;
  const m = Math.floor(diff / 60000);
  if (m < 1) return "just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

// ── TypingDots ──────────────────────────────────────────────────────────────

function TypingDots() {
  return (
    <span className="inline-flex gap-0.5 ml-1 align-middle">
      {[0, 1, 2].map((i) => (
        <span
          key={i}
          className="w-1 h-1 rounded-full bg-muted-foreground animate-bounce"
          style={{ animationDelay: `${i * 150}ms` }}
        />
      ))}
    </span>
  );
}

// ── Main component ──────────────────────────────────────────────────────────

const MIN_HEIGHT = 200;
const MAX_HEIGHT_VH = 0.7;

export default function CopilotBottomPanel() {
  const navigate = useNavigate();
  const {
    context,
    open,
    setOpen,
    panelHeight,
    setPanelHeight,
    sessions,
    activeSessionId,
    activeMessages,
    createSession,
    selectSession,
    updateActiveMessages,
  } = useCopilotContextValue();

  const { send, isStreaming } = useChatStreamSession();
  const [input, setInput] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [sessionPopoverOpen, setSessionPopoverOpen] = useState(false);

  // Auto-scroll
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [activeMessages]);

  // Focus input when panel opens
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open]);

  // ⌘K / Ctrl+K global shortcut
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      // Escape closes
      if (e.key === "Escape" && open) {
        setOpen(false);
        return;
      }
      // ⌘K / Ctrl+K toggles
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        const tag = (document.activeElement as HTMLElement)?.tagName ?? "";
        const isEditable =
          tag === "INPUT" ||
          tag === "TEXTAREA" ||
          (document.activeElement as HTMLElement)?.isContentEditable;
        if (!isEditable) {
          e.preventDefault();
          setOpen(!open);
        }
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [open, setOpen]);

  // Resize handle drag
  const dragging = useRef(false);
  const dragStartY = useRef(0);
  const dragStartH = useRef(0);

  const onResizePointerDown = useCallback((e: React.PointerEvent) => {
    dragging.current = true;
    dragStartY.current = e.clientY;
    dragStartH.current = panelHeight;
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
  }, [panelHeight]);

  const onResizePointerMove = useCallback((e: React.PointerEvent) => {
    if (!dragging.current) return;
    const delta = dragStartY.current - e.clientY;
    const maxH = window.innerHeight * MAX_HEIGHT_VH;
    const newH = Math.min(maxH, Math.max(MIN_HEIGHT, dragStartH.current + delta));
    setPanelHeight(newH);
  }, [setPanelHeight]);

  const onResizePointerUp = useCallback(() => {
    dragging.current = false;
  }, []);

  // Send message
  async function handleSend() {
    const text = input.trim();
    if (!text || isStreaming) return;
    setInput("");

    // Ensure a session exists
    let sessionId = activeSessionId;
    if (!sessionId) {
      sessionId = createSession();
    }

    await send(text, context, activeMessages, (msgs) => {
      updateActiveMessages(msgs);
    });

    // After stream ends, execute guidance actions on final assistant message
    // (handled in useEffect below watching activeMessages)
  }

  // Execute guidance actions when streaming ends
  const prevStreamingRef = useRef(false);
  useEffect(() => {
    const lastMsg = activeMessages[activeMessages.length - 1];
    const wasStreaming = prevStreamingRef.current;
    const nowDone = lastMsg?.role === "assistant" && !lastMsg.streaming;

    if (wasStreaming && nowDone && lastMsg) {
      const { navPath, highlights } = parseGuidanceActions(lastMsg.content);
      if (navPath) navigate(navPath);
      highlights.forEach(applyHighlight);

      // Clean up tokens from stored message
      const cleaned = parseGuidanceActions(lastMsg.content).cleaned;
      if (cleaned !== lastMsg.content) {
        const updated = activeMessages.map((m, i) =>
          i === activeMessages.length - 1 ? { ...m, content: cleaned } : m
        );
        updateActiveMessages(updated);
      }
    }
    prevStreamingRef.current = !!lastMsg?.streaming;
  }, [activeMessages, navigate, updateActiveMessages]);

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  function handleNewChat() {
    createSession();
    setSessionPopoverOpen(false);
  }

  if (!open) return null;

  return (
    <div
      className="fixed bottom-0 left-0 right-0 z-50 flex flex-col bg-background border-t border-border shadow-2xl"
      style={{ height: panelHeight }}
    >
      {/* Resize handle */}
      <div
        className="h-1.5 w-full cursor-row-resize flex items-center justify-center shrink-0 hover:bg-primary/20 transition-colors"
        onPointerDown={onResizePointerDown}
        onPointerMove={onResizePointerMove}
        onPointerUp={onResizePointerUp}
      >
        <div className="w-8 h-0.5 rounded-full bg-border" />
      </div>

      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-1.5 border-b shrink-0">
        <span className="text-xs font-semibold uppercase tracking-widest">Copilot</span>
        {context && (
          <span className="text-xs text-muted-foreground truncate max-w-[300px]">
            — {context.type}: {context.label}
          </span>
        )}
        <div className="ml-auto flex items-center gap-1">
          {/* Session dropdown */}
          <Popover open={sessionPopoverOpen} onOpenChange={setSessionPopoverOpen}>
            <PopoverTrigger asChild>
              <Button size="sm" variant="ghost" className="h-6 text-xs px-2">
                {sessions.length > 0
                  ? (sessions.find((s) => s.id === activeSessionId)?.title ?? "New chat")
                  : "New chat"}
                {" ▾"}
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-64 p-1" align="end">
              <button
                className="w-full text-left text-xs px-2 py-1.5 hover:bg-muted rounded font-medium"
                onClick={handleNewChat}
              >
                + New chat
              </button>
              {sessions.length > 0 && (
                <div className="border-t mt-1 pt-1">
                  {[...sessions].reverse().map((s) => (
                    <button
                      key={s.id}
                      className={`w-full text-left text-xs px-2 py-1.5 rounded flex justify-between items-center gap-2 ${
                        s.id === activeSessionId ? "bg-muted" : "hover:bg-muted/50"
                      }`}
                      onClick={() => {
                        selectSession(s.id);
                        setSessionPopoverOpen(false);
                      }}
                    >
                      <span className="truncate flex-1">{s.title}</span>
                      <span className="text-muted-foreground shrink-0">{relativeTime(s.createdAt)}</span>
                    </button>
                  ))}
                </div>
              )}
            </PopoverContent>
          </Popover>

          <Button
            size="sm"
            variant="ghost"
            className="h-6 w-6 p-0 text-muted-foreground"
            onClick={() => setOpen(false)}
          >
            ✕
          </Button>
        </div>
      </div>

      {/* Message list */}
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3 min-h-0">
        {activeMessages.length === 0 && (
          <div className="flex h-full items-center justify-center text-muted-foreground text-xs">
            {context ? `Ask anything about ${context.label}` : "Ask anything about the platform"}
          </div>
        )}
        {activeMessages.map((msg, i) => (
          <div key={i} className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}>
            {msg.role === "user" ? (
              <span className="max-w-[80%] rounded-2xl px-3 py-1.5 text-xs bg-primary text-primary-foreground">
                {msg.content}
              </span>
            ) : (
              <span className="max-w-[85%] text-xs whitespace-pre-wrap">
                {parseGuidanceActions(msg.content).cleaned || msg.content}
                {msg.streaming && <TypingDots />}
              </span>
            )}
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      {/* Input row */}
      <div className="border-t px-3 py-2 flex gap-2 shrink-0">
        <Input
          ref={inputRef}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Ask a question… (⌘K to close)"
          className="text-xs h-7"
          disabled={isStreaming}
        />
        <Button size="sm" className="h-7" onClick={handleSend} disabled={isStreaming || !input.trim()}>
          Send
        </Button>
      </div>
    </div>
  );
}
