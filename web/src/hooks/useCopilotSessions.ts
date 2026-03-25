import { useState, useCallback } from "react";
import type { Message } from "./useChatStream";

export interface ChatSession {
  id: string;
  createdAt: number;
  title: string; // first user message, truncated to 40 chars
  messages: Message[];
}

const STORAGE_KEY = "unc:copilot:sessions";
const MAX_SESSIONS = 20;

function genId(): string {
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
}

function loadSessions(): ChatSession[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    return JSON.parse(raw) as ChatSession[];
  } catch {
    return [];
  }
}

function saveSessions(sessions: ChatSession[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(sessions));
  } catch {
    // Storage full — ignore
  }
}

export function useCopilotSessions() {
  const [sessions, setSessions] = useState<ChatSession[]>(() => loadSessions());
  const [activeSessionId, setActiveSessionId] = useState<string | null>(() => {
    const saved = loadSessions();
    return saved.length > 0 ? saved[saved.length - 1].id : null;
  });

  const activeSession = sessions.find((s) => s.id === activeSessionId) ?? null;

  const persistAndSet = useCallback((next: ChatSession[]) => {
    // Prune oldest if over limit
    const pruned = next.length > MAX_SESSIONS ? next.slice(next.length - MAX_SESSIONS) : next;
    saveSessions(pruned);
    setSessions(pruned);
  }, []);

  const createSession = useCallback((): string => {
    const id = genId();
    const session: ChatSession = {
      id,
      createdAt: Date.now(),
      title: "New chat",
      messages: [],
    };
    setSessions((prev) => {
      const next = [...prev, session];
      const pruned = next.length > MAX_SESSIONS ? next.slice(next.length - MAX_SESSIONS) : next;
      saveSessions(pruned);
      return pruned;
    });
    setActiveSessionId(id);
    return id;
  }, []);

  const updateSession = useCallback((id: string, messages: Message[]) => {
    setSessions((prev) => {
      const next = prev.map((s) => {
        if (s.id !== id) return s;
        // Auto-title from first user message
        const firstUser = messages.find((m) => m.role === "user");
        const title = firstUser
          ? firstUser.content.slice(0, 40) + (firstUser.content.length > 40 ? "…" : "")
          : s.title;
        return { ...s, title, messages };
      });
      saveSessions(next);
      return next;
    });
  }, []);

  const selectSession = useCallback((id: string) => {
    setActiveSessionId(id);
  }, []);

  return {
    sessions,
    activeSessionId,
    activeSession,
    createSession,
    updateSession,
    selectSession,
    persistAndSet,
  };
}
