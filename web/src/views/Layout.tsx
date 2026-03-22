import { Outlet } from "react-router-dom";
import { useThemeNew, type ColorMode } from "../hooks/useThemeNew";

const MODE_ICONS: Record<string, string> = {
  light: "\u2600",    // ☀
  dark: "\u263E",     // ☾
  system: "\u25D1",   // ◑
};

const MODE_CYCLE: ColorMode[] = ["light", "dark", "system"];

/**
 * Root layout — renders the current route with theme toggle.
 */
export default function Layout() {
  const { mode, setMode, resolvedTheme } = useThemeNew();

  function cycleMode() {
    const idx = MODE_CYCLE.indexOf(mode);
    const next = MODE_CYCLE[(idx + 1) % MODE_CYCLE.length];
    setMode(next);
  }

  return (
    <div className="h-screen w-screen overflow-hidden bg-background text-foreground font-mono text-sm flex flex-col">
      <div className="flex-1 min-h-0">
        <Outlet />
      </div>
      <div className="flex items-center justify-between border-t px-4 py-1">
        <span className="text-[10px] text-muted-foreground">UNCWORKS</span>
        <button
          onClick={cycleMode}
          className="flex items-center gap-1.5 px-2 py-0.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
          title={`Theme: ${mode} (${resolvedTheme})`}
        >
          <span>{MODE_ICONS[mode]}</span>
          <span className="text-[10px] uppercase">{mode}</span>
        </button>
      </div>
    </div>
  );
}
