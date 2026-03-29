import { createContext, useContext, useState, useEffect, useRef, useCallback } from "react";
import type { ReactNode } from "react";
import type { ChatContext, Message } from "./useChatStream";
import { useCopilotSessions } from "./useCopilotSessions";
import type { ChatSession } from "./useCopilotSessions";

interface CopilotContextValue {
  // Page context (registered by views)
  context: ChatContext | null;
  setContext: (ctx: ChatContext | null) => void;

  // Panel open/resize state
  open: boolean;
  setOpen: (open: boolean) => void;
  panelHeight: number;
  setPanelHeight: (h: number) => void;

  // Session management
  sessions: ChatSession[];
  activeSessionId: string | null;
  activeMessages: Message[];
  createSession: () => string;
  selectSession: (id: string) => void;
  updateActiveMessages: (messages: Message[], sessionIdOverride?: string) => void;
}

const CopilotContext = createContext<CopilotContextValue>({
  context: null,
  setContext: () => {},
  open: false,
  setOpen: () => {},
  panelHeight: 320,
  setPanelHeight: () => {},
  sessions: [],
  activeSessionId: null,
  activeMessages: [],
  createSession: () => "",
  selectSession: () => {},
  updateActiveMessages: (_msgs, _id) => {},
});

export function CopilotContextProvider({ children }: { children: ReactNode }) {
  const [context, setContext] = useState<ChatContext | null>(null);
  const [open, setOpen] = useState(false);
  const [panelHeight, setPanelHeight] = useState(320);

  const { sessions, activeSessionId, activeSession, createSession, updateSession, selectSession } =
    useCopilotSessions();

  const updateActiveMessages = useCallback(
    (messages: Message[], sessionIdOverride?: string) => {
      const id = sessionIdOverride ?? activeSessionId;
      if (id) {
        updateSession(id, messages);
      }
    },
    [activeSessionId, updateSession]
  );

  const activeMessages = activeSession?.messages ?? [];

  return (
    <CopilotContext.Provider
      value={{
        context,
        setContext,
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
      }}
    >
      {children}
    </CopilotContext.Provider>
  );
}

export function useCopilotContextValue() {
  return useContext(CopilotContext);
}

/**
 * Call in a view to register page-level context for the global CopilotPanel.
 * Clears the context automatically on unmount.
 */
export function useCopilotContext(ctx: ChatContext | null) {
  const { setContext } = useContext(CopilotContext);
  const serialized = JSON.stringify(ctx);
  const prevRef = useRef<string | null>(null);

  useEffect(() => {
    if (prevRef.current !== serialized) {
      prevRef.current = serialized;
      // ctx and serialized are always in sync — parse only when value changed
      setContext(ctx);
    }
    // ctx intentionally omitted: serialized covers its value identity
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serialized, setContext]);

  useEffect(() => {
    return () => setContext(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
}
