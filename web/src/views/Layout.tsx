import { Outlet } from "react-router-dom";
import { useThemeNew } from "../hooks/useThemeNew";

/**
 * Root layout — renders the current route with theme toggle.
 */
export default function Layout() {
  const { mode, toggleMode } = useThemeNew();

  const resolvedMode =
    mode === "system"
      ? window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light"
      : mode;

  return (
    <div className="h-screen w-screen overflow-hidden bg-background text-foreground font-mono text-sm flex flex-col">
      <div className="flex-1 min-h-0">
        <Outlet />
      </div>
      <div className="flex items-center justify-end border-t px-4 py-1">
        <button
          onClick={toggleMode}
          className="px-1.5 py-0.5 text-sm text-muted-foreground hover:text-foreground"
          title={`Switch to ${resolvedMode === "dark" ? "light" : "dark"} mode`}
        >
          {resolvedMode === "dark" ? "\u2600" : "\u263E"}
        </button>
      </div>
    </div>
  );
}
