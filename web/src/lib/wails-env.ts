// wails-env.ts — Detect and configure Wails desktop environment.
// Sets data-wails on <html> for CSS targeting and installs native-app UX fixes.

declare global {
  interface Window {
    go?: unknown;
    runtime?: unknown;
  }
}

export function isWails(): boolean {
  return typeof window.go !== "undefined" || typeof window.runtime !== "undefined";
}

export function isMac(): boolean {
  return navigator.platform.startsWith("Mac") || navigator.userAgent.includes("Mac");
}

// suppressContextMenu installs a contextmenu handler that allows the native
// text-editing menu on inputs/textareas/selections, and blocks everything else.
function suppressContextMenu() {
  document.addEventListener("contextmenu", (e) => {
    const target = e.target as HTMLElement;
    if (
      target instanceof HTMLInputElement ||
      target instanceof HTMLTextAreaElement ||
      target.isContentEditable
    ) return;
    const sel = window.getSelection();
    if (sel && sel.toString().length > 0) return;
    e.preventDefault();
  });
}

const FONT_SIZE_KEY = "unc-font-size";
const FONT_STEP = 1; // px per step
const FONT_MIN = 10;
const FONT_MAX = 24;
const FONT_DEFAULT = 13;

function applyFontSize(px: number) {
  document.documentElement.style.fontSize = `${px}px`;
  localStorage.setItem(FONT_SIZE_KEY, String(px));
}

function loadFontSize() {
  const v = localStorage.getItem(FONT_SIZE_KEY);
  return v ? Math.max(FONT_MIN, Math.min(FONT_MAX, Number(v))) : FONT_DEFAULT;
}

function installFontScaling() {
  applyFontSize(loadFontSize());

  document.addEventListener("keydown", (e) => {
    if (!e.metaKey) return;
    const current = loadFontSize();
    if (e.key === "=" || e.key === "+") {
      e.preventDefault();
      applyFontSize(Math.min(FONT_MAX, current + FONT_STEP));
    } else if (e.key === "-") {
      e.preventDefault();
      applyFontSize(Math.max(FONT_MIN, current - FONT_STEP));
    } else if (e.key === "0") {
      e.preventDefault();
      applyFontSize(FONT_DEFAULT);
    } else if (e.key === ",") {
      // Cmd+, — open Preferences/Settings
      e.preventDefault();
      window.dispatchEvent(new CustomEvent("uncworks:open-settings"));
    }
  });
}

export function setupWailsEnv() {
  if (!isWails()) return;

  document.documentElement.setAttribute("data-wails", "true");
  suppressContextMenu();
  installFontScaling();
}
