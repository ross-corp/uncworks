import { Outlet } from "react-router-dom";

/**
 * Root layout — just renders the current route.
 * No sidebar, no header chrome. Full screen for each view.
 */
export default function Layout() {
  return (
    <div className="h-screen w-screen overflow-hidden bg-background text-foreground font-mono text-sm">
      <Outlet />
    </div>
  );
}
