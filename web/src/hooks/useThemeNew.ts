import { useState, useEffect, useCallback } from "react";

export type ColorMode = "light" | "dark" | "system";

const THEME_KEY = "aot-theme-color";
const MODE_KEY = "aot-theme-mode";

// shadcn built-in themes (maps to CSS class on <html>)
export const THEMES = [
  "zinc", "slate", "stone", "gray", "neutral",
  "red", "rose", "orange", "green", "blue", "yellow", "violet",
] as const;

export type Theme = typeof THEMES[number];

function getSystemMode(): "light" | "dark" {
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyTheme(theme: Theme, mode: ColorMode) {
  const root = document.documentElement;

  // Remove all theme classes
  THEMES.forEach((t) => root.classList.remove(`theme-${t}`));

  // Apply theme class (zinc is the default, no class needed)
  if (theme !== "zinc") {
    root.classList.add(`theme-${theme}`);
  }

  // Apply dark/light
  const resolvedMode = mode === "system" ? getSystemMode() : mode;
  root.classList.toggle("dark", resolvedMode === "dark");
}

export function useThemeNew() {
  const [theme, setThemeState] = useState<Theme>(() => {
    return (localStorage.getItem(THEME_KEY) as Theme) || "zinc";
  });

  const [mode, setModeState] = useState<ColorMode>(() => {
    return (localStorage.getItem(MODE_KEY) as ColorMode) || "system";
  });

  // Apply on mount and changes
  useEffect(() => {
    applyTheme(theme, mode);
  }, [theme, mode]);

  // Listen for system theme changes
  useEffect(() => {
    if (mode !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => applyTheme(theme, "system");
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [theme, mode]);

  const setTheme = useCallback((t: Theme) => {
    setThemeState(t);
    localStorage.setItem(THEME_KEY, t);
  }, []);

  const setMode = useCallback((m: ColorMode) => {
    setModeState(m);
    localStorage.setItem(MODE_KEY, m);
  }, []);

  const toggleMode = useCallback(() => {
    const resolvedCurrent = mode === "system" ? getSystemMode() : mode;
    const next = resolvedCurrent === "dark" ? "light" : "dark";
    setMode(next);
  }, [mode, setMode]);

  return { theme, mode, setTheme, setMode, toggleMode, themes: THEMES };
}
