/**
 * Like useChatStream but messages state is externally owned (via session store).
 * The caller provides the current messages and an updater function.
 */
import { useState, useCallback, useRef, useEffect } from "react";
import { toast } from "sonner";
import { apiFetch } from "./apiFetch";
import type { ChatContext, Message } from "./useChatStream";

interface UseChatStreamSessionReturn {
  send: (userText: string, context: ChatContext | null, currentMessages: Message[], onUpdate: (msgs: Message[]) => void) => Promise<void>;
  isStreaming: boolean;
}

export function useChatStreamSession(): UseChatStreamSessionReturn {
  const [isStreaming, setIsStreaming] = useState(false);
  const abortRef = useRef<AbortController | null>(null);
  const mountedRef = useRef(true);

  // Abort any in-flight stream when the hook's owning component unmounts.
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      abortRef.current?.abort();
    };
  }, []);

  const send = useCallback(async (
    userText: string,
    context: ChatContext | null,
    currentMessages: Message[],
    onUpdate: (msgs: Message[]) => void,
  ) => {
    if (!userText.trim() || isStreaming) return;

    const userMessage: Message = { role: "user", content: userText };
    const assistantPlaceholder: Message = { role: "assistant", content: "", streaming: true };

    const withUser = [...currentMessages, userMessage, assistantPlaceholder];
    onUpdate(withUser);
    if (mountedRef.current) setIsStreaming(true);

    // History sent to API: all messages except the streaming placeholder
    const history = [...currentMessages, userMessage];

    const ac = new AbortController();
    abortRef.current = ac;

    try {
      const resp = await apiFetch("/api/v1/chat/stream", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        signal: ac.signal,
        body: JSON.stringify({
          messages: history.map((m) => ({ role: m.role, content: m.content })),
          context: context ?? undefined,
        }),
      });

      if (!resp.ok || !resp.body) {
        throw new Error(`HTTP ${resp.status}`);
      }

      const reader = resp.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      let accumulated = "";

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
            const finalMsgs = withUser.slice(0, -1).concat({
              role: "assistant",
              content: accumulated,
              streaming: false,
            });
            if (mountedRef.current) {
              onUpdate(finalMsgs);
              setIsStreaming(false);
            }
            return;
          }

          try {
            const parsed = JSON.parse(payload);
            if (parsed.error) {
              toast.error(`Chat error: ${parsed.error}`);
              if (mountedRef.current) {
                onUpdate(currentMessages); // revert to before user message
                setIsStreaming(false);
              }
              return;
            }
            const delta =
              parsed?.choices?.[0]?.delta?.content ??
              parsed?.choices?.[0]?.delta?.reasoning_content;
            if (typeof delta === "string") {
              accumulated += delta;
              if (mountedRef.current) {
                const streamingMsgs = withUser.slice(0, -1).concat({
                  role: "assistant",
                  content: accumulated,
                  streaming: true,
                });
                onUpdate(streamingMsgs);
              }
            }
          } catch {
            // Non-JSON line — skip
          }
        }
      }

      // Stream ended without [DONE]
      if (mountedRef.current) {
        const finalMsgs = withUser.slice(0, -1).concat({
          role: "assistant",
          content: accumulated,
          streaming: false,
        });
        onUpdate(finalMsgs);
      }
    } catch (err: unknown) {
      if (err instanceof Error && err.name === "AbortError") return;
      toast.error("Chat unavailable");
      if (mountedRef.current) onUpdate(currentMessages);
    } finally {
      if (mountedRef.current) setIsStreaming(false);
    }
  }, [isStreaming]);

  return { send, isStreaming };
}
