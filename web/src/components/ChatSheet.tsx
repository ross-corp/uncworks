import { useRef, useEffect, useState } from "react";
import {
  Sheet, SheetContent, SheetHeader, SheetTitle,
} from "./ui/sheet";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { useChatStream } from "../hooks/useChatStream";
import type { ChatContext } from "../hooks/useChatStream";

interface ChatSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  context?: ChatContext;
  title?: string;
}

export default function ChatSheet({ open, onOpenChange, context, title = "Chat" }: ChatSheetProps) {
  const { messages, send, isStreaming, reset } = useChatStream();
  const [input, setInput] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when messages change.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  function handleClose(o: boolean) {
    if (!o) reset();
    onOpenChange(o);
  }

  async function handleSend() {
    const text = input.trim();
    if (!text || isStreaming) return;
    setInput("");
    await send(text, context);
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent side="right" className="w-[420px] flex flex-col p-0 gap-0">
        <SheetHeader className="px-4 py-3 border-b shrink-0">
          <SheetTitle className="text-sm font-semibold">{title}</SheetTitle>
        </SheetHeader>

        {/* Message list */}
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
          {messages.length === 0 && (
            <div className="flex h-full items-center justify-center text-muted-foreground text-xs">
              Ask anything about this spec
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
          />
          <Button size="sm" onClick={handleSend} disabled={isStreaming || !input.trim()}>
            Send
          </Button>
        </div>
      </SheetContent>
    </Sheet>
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
