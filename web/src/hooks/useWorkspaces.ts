import { useState, useCallback } from "react";
import type { Repository } from "../types/agent-run";

export interface Workspace {
  id: string;
  name: string;
  description: string;
  repos: Repository[];
}

const STORAGE_KEY = "uncworks:workspaces";

function loadWorkspaces(): Workspace[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

function saveWorkspaces(ws: Workspace[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(ws));
}

export function useWorkspaces() {
  const [workspaces, setWorkspaces] = useState<Workspace[]>(loadWorkspaces);

  const addWorkspace = useCallback((ws: Omit<Workspace, "id">) => {
    setWorkspaces((prev) => {
      const next = [...prev, { ...ws, id: crypto.randomUUID() }];
      saveWorkspaces(next);
      return next;
    });
  }, []);

  const updateWorkspace = useCallback((id: string, updates: Partial<Omit<Workspace, "id">>) => {
    setWorkspaces((prev) => {
      const next = prev.map((w) => (w.id === id ? { ...w, ...updates } : w));
      saveWorkspaces(next);
      return next;
    });
  }, []);

  const deleteWorkspace = useCallback((id: string) => {
    setWorkspaces((prev) => {
      const next = prev.filter((w) => w.id !== id);
      saveWorkspaces(next);
      return next;
    });
  }, []);

  return { workspaces, addWorkspace, updateWorkspace, deleteWorkspace };
}
