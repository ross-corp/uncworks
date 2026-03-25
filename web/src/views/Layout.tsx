import { Outlet } from "react-router-dom";
import GlobalNav from "../components/GlobalNav";

export default function Layout() {
  return (
    <div className="h-screen w-screen overflow-hidden bg-background text-foreground font-mono text-sm flex flex-row">
      <GlobalNav />
      <main className="flex-1 min-w-0 flex flex-col h-screen overflow-hidden">
        <Outlet />
      </main>
    </div>
  );
}
