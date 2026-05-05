import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import AppNew from "./AppNew";
import ErrorBoundary from "./components/ErrorBoundary";
import { ToastProvider } from "./components/Toast";
import { setupWailsEnv } from "./lib/wails-env";
import "./index.css";

setupWailsEnv();

// Bridge Wails runtime events → DOM custom events so React components
// can listen without importing the Wails runtime directly.
if (typeof window !== "undefined") {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const w = window as any;
  // Wails runtime is injected asynchronously; poll until it's available.
  const bridgeWailsEvents = () => {
    if (!w.runtime?.EventsOn) return;
    w.runtime.EventsOn("app:open-settings", () => {
      window.dispatchEvent(new CustomEvent("uncworks:open-settings"));
    });
    w.runtime.EventsOn("app:local-reload", () => {
      window.dispatchEvent(new CustomEvent("uncworks:local-reload"));
    });
    w.runtime.EventsOn("settings:changed", () => {
      window.dispatchEvent(new CustomEvent("uncworks:settings-changed"));
    });
  };
  // Try immediately and again after DOMContentLoaded
  bridgeWailsEvents();
  window.addEventListener("DOMContentLoaded", bridgeWailsEvents);
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ErrorBoundary>
      <ToastProvider>
        <AppNew />
      </ToastProvider>
    </ErrorBoundary>
  </StrictMode>
);
