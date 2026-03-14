import { useState, useCallback, useMemo } from "react";

const STORAGE_KEY = "uncworks:repos";

function loadRepos(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

function saveRepos(repos: string[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(repos));
}

/**
 * Manages a localStorage-backed registry of repo URLs.
 * Accepts `runDerivedRepos` to merge with the persisted list for display.
 */
export function useRepoRegistry(runDerivedRepos: string[] = []) {
  const [registryRepos, setRegistryRepos] = useState<string[]>(loadRepos);

  const addRepo = useCallback((url: string) => {
    setRegistryRepos((prev) => {
      if (prev.includes(url)) return prev;
      const next = [...prev, url];
      saveRepos(next);
      return next;
    });
  }, []);

  const removeRepo = useCallback((url: string) => {
    setRegistryRepos((prev) => {
      const next = prev.filter((r) => r !== url);
      saveRepos(next);
      return next;
    });
  }, []);

  /** Deduplicated union of registry + run-derived repos. */
  const repos = useMemo(() => {
    const set = new Set([...registryRepos, ...runDerivedRepos]);
    return [...set];
  }, [registryRepos, runDerivedRepos]);

  return { repos, registryRepos, addRepo, removeRepo };
}
