import { Outlet } from "react-router-dom";
import { useThemeNew, type ColorMode } from "../hooks/useThemeNew";

const MODE_CYCLE: ColorMode[] = ["light", "dark", "system"];

export default function Layout() {
  const { mode, setMode } = useThemeNew();

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
        <span className="text-xs text-muted-foreground tracking-wider">UNCWORKS</span>
        <button
          onClick={cycleMode}
          className="text-xs text-muted-foreground hover:text-foreground transition-colors px-2 py-0.5 rounded-md"
          title={`Theme: ${mode}`}
        >
          {mode}
        </button>
      </div>
    </div>
  );
}
