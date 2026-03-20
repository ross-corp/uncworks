import { useState, useEffect, useCallback } from "react";

export type ColorMode = "light" | "dark" | "system";

const MODE_KEY = "aot-theme-mode";

function getSystemMode(): "light" | "dark" {
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyMode(mode: ColorMode) {
  const root = document.documentElement;
  const resolvedMode = mode === "system" ? getSystemMode() : mode;
  root.classList.toggle("dark", resolvedMode === "dark");
}

export function useThemeNew() {
  const [mode, setModeState] = useState<ColorMode>(() => {
    return (localStorage.getItem(MODE_KEY) as ColorMode) || "system";
  });

  // Apply on mount and changes
  useEffect(() => {
    applyMode(mode);
  }, [mode]);

  // Listen for system theme changes
  useEffect(() => {
    if (mode !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => applyMode("system");
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [mode]);

  const setMode = useCallback((m: ColorMode) => {
    setModeState(m);
    localStorage.setItem(MODE_KEY, m);
  }, []);

  const toggleMode = useCallback(() => {
    const resolvedCurrent = mode === "system" ? getSystemMode() : mode;
    const next = resolvedCurrent === "dark" ? "light" : "dark";
    setMode(next);
  }, [mode, setMode]);

  return { mode, setMode, toggleMode };
}
