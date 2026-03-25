import { useEffect, useRef, useState } from "react";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
} from "./ui/dialog";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { useChatStream } from "../hooks/useChatStream";
import { useCopilotContextValue } from "../hooks/useCopilotContext";

export default function CopilotPanel() {
  const [open, setOpen] = useState(false);
  const { context } = useCopilotContextValue();
  const { messages, send, isStreaming, reset } = useChatStream();
  const [input, setInput] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);

  // Global ⌘K / Ctrl+K shortcut.
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      const tag = (document.activeElement as HTMLElement)?.tagName ?? "";
      const isEditable =
        tag === "INPUT" ||
        tag === "TEXTAREA" ||
        (document.activeElement as HTMLElement)?.isContentEditable;

      if ((e.metaKey || e.ctrlKey) && e.key === "k" && !isEditable) {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, []);

  // Auto-scroll to bottom.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  function handleClose(o: boolean) {
    if (!o) reset();
    setOpen(o);
  }

  async function handleSend() {
    const text = input.trim();
    if (!text || isStreaming) return;
    setInput("");
    await send(text, context ?? undefined);
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-[600px] w-full flex flex-col gap-0 p-0 h-[520px]">
        <DialogHeader className="px-4 py-3 border-b shrink-0">
          <DialogTitle className="text-sm font-semibold flex items-center gap-2">
            Copilot
            {context && (
              <span className="text-xs font-normal text-muted-foreground truncate max-w-[360px]">
                — {context.type}: {context.label}
              </span>
            )}
          </DialogTitle>
        </DialogHeader>

        {/* Message list */}
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3 min-h-0">
          {messages.length === 0 && (
            <div className="flex h-full items-center justify-center text-muted-foreground text-xs">
              {context
                ? `Ask anything about ${context.label}`
                : "Ask anything about the platform"}
            </div>
          )}
          {messages.map((msg, i) => (
            <div key={i} className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}>
              {msg.role === "user" ? (
                <span className="max-w-[80%] rounded-2xl px-3 py-1.5 text-sm bg-primary text-primary-foreground">
                  {msg.content}
                </span>
              ) : (
                <span className="max-w-[85%] text-sm whitespace-pre-wrap">
                  {msg.content}
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
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask a question..."
            className="text-sm h-8"
            disabled={isStreaming}
            autoFocus
          />
          <Button size="sm" onClick={handleSend} disabled={isStreaming || !input.trim()}>
            Send
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

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
