import { useState, useCallback } from "react";
import { toast } from "sonner";

export interface Message {
  role: "user" | "assistant";
  content: string;
  streaming?: boolean;
}

export interface ChatContext {
  type: "spec" | "run" | "project" | "general";
  content: string;
  label: string;
}

interface UseChatStreamReturn {
  messages: Message[];
  send: (userText: string, context?: ChatContext) => Promise<void>;
  isStreaming: boolean;
  reset: () => void;
}

export function useChatStream(): UseChatStreamReturn {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);

  const reset = useCallback(() => {
    setMessages([]);
    setIsStreaming(false);
  }, []);

  const send = useCallback(async (userText: string, context?: ChatContext) => {
    if (!userText.trim() || isStreaming) return;

    const userMessage: Message = { role: "user", content: userText };
    const assistantMessage: Message = { role: "assistant", content: "", streaming: true };

    setMessages((prev) => [...prev, userMessage, assistantMessage]);
    setIsStreaming(true);

    // Build conversation history for the API (exclude the streaming placeholder).
    const history: Message[] = [...messages, userMessage];

    try {
      const resp = await fetch("/api/v1/chat/stream", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          messages: history.map((m) => ({ role: m.role, content: m.content })),
          context,
        }),
      });

      if (!resp.ok || !resp.body) {
        throw new Error(`HTTP ${resp.status}`);
      }

      const reader = resp.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() ?? "";

        for (const line of lines) {
          if (!line.startsWith("data: ")) continue;
          const payload = line.slice(6).trim();

          if (payload === "[DONE]") {
            setMessages((prev) => {
              const next = [...prev];
              const last = next[next.length - 1];
              if (last?.role === "assistant") {
                next[next.length - 1] = { ...last, streaming: false };
              }
              return next;
            });
            setIsStreaming(false);
            return;
          }

          // Check for error payload.
          try {
            const parsed = JSON.parse(payload);
            if (parsed.error) {
              toast.error(`Chat error: ${parsed.error}`);
              setMessages((prev) => prev.slice(0, -1)); // remove streaming placeholder
              setIsStreaming(false);
              return;
            }
            // Extract delta content.
            const delta = parsed?.choices?.[0]?.delta?.content;
            if (typeof delta === "string") {
              setMessages((prev) => {
                const next = [...prev];
                const last = next[next.length - 1];
                if (last?.role === "assistant") {
                  next[next.length - 1] = { ...last, content: last.content + delta };
                }
                return next;
              });
            }
          } catch {
            // Non-JSON line — skip
          }
        }
      }

      // Stream ended without [DONE] — mark complete anyway.
      setMessages((prev) => {
        const next = [...prev];
        const last = next[next.length - 1];
        if (last?.role === "assistant") {
          next[next.length - 1] = { ...last, streaming: false };
        }
        return next;
      });
    } catch (err) {
      toast.error("Chat unavailable");
      setMessages((prev) => prev.slice(0, -1)); // remove streaming placeholder
    } finally {
      setIsStreaming(false);
    }
  }, [messages, isStreaming]);

  return { messages, send, isStreaming, reset };
}
