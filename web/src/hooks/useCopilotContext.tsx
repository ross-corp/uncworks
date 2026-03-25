import { createContext, useContext, useState, useEffect, useRef } from "react";
import type { ReactNode } from "react";
import type { ChatContext } from "./useChatStream";

interface CopilotContextValue {
  context: ChatContext | null;
  setContext: (ctx: ChatContext | null) => void;
}

const CopilotContext = createContext<CopilotContextValue>({
  context: null,
  setContext: () => {},
});

export function CopilotContextProvider({ children }: { children: ReactNode }) {
  const [context, setContext] = useState<ChatContext | null>(null);
  return (
    <CopilotContext.Provider value={{ context, setContext }}>
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
      setContext(ctx);
    }
  }, [serialized, setContext, ctx]);

  useEffect(() => {
    return () => setContext(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
}
