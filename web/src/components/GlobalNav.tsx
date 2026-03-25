import { useState, useEffect } from "react";
import { Link, useLocation } from "react-router-dom";
import { useThemeNew, type ColorMode } from "../hooks/useThemeNew";
import { apiFetch } from "../hooks/apiFetch";

const MODE_CYCLE: ColorMode[] = ["light", "dark", "system"];

interface NavItem {
  label: string;
  path: string;
  icon: string;
  countKey: string;
}

const NAV_ITEMS: NavItem[] = [
  { label: "Runs", path: "/", icon: "▶", countKey: "runs" },
  { label: "Projects", path: "/projects", icon: "◈", countKey: "projects" },
  { label: "Templates", path: "/templates", icon: "◻", countKey: "templates" },
  { label: "Chains", path: "/chains", icon: "⛓", countKey: "chains" },
  { label: "Schedules", path: "/schedules", icon: "⏱", countKey: "schedules" },
];

interface Counts {
  runs: number | null;
  projects: number | null;
  templates: number | null;
  chains: number | null;
  chainruns: number | null;
  schedules: number | null;
}

export default function GlobalNav() {
  const location = useLocation();
  const { mode, setMode } = useThemeNew();

  const [collapsed, setCollapsed] = useState<boolean>(() => {
    return localStorage.getItem("nav-collapsed") === "true";
  });

  const [counts, setCounts] = useState<Counts>({
    runs: null,
    projects: null,
    templates: null,
    chains: null,
    chainruns: null,
    schedules: null,
  });

  function toggleCollapsed() {
    setCollapsed((prev) => {
      const next = !prev;
      localStorage.setItem("nav-collapsed", String(next));
      return next;
    });
  }

  function cycleMode() {
    const idx = MODE_CYCLE.indexOf(mode);
    const next = MODE_CYCLE[(idx + 1) % MODE_CYCLE.length];
    setMode(next);
  }

  useEffect(() => {
    let cancelled = false;

    const fetch = async () => {
      try {
        const [runsResp, projectsResp, templatesResp, chainsResp, chainrunsResp, schedulesResp] = await Promise.allSettled([
          apiFetch("/api/v1/runs"),
          apiFetch("/api/v1/projects"),
          apiFetch("/api/v1/templates"),
          apiFetch("/api/v1/chains"),
          apiFetch("/api/v1/chainruns"),
          apiFetch("/api/v1/schedules"),
        ]);

        let runsCount: number | null = null;
        if (runsResp.status === "fulfilled" && runsResp.value.ok) {
          const data = await runsResp.value.json();
          const items: { status?: { phase?: string } }[] = Array.isArray(data) ? data : (data.items || []);
          runsCount = items.filter((r) => {
            const phase = r.status?.phase;
            return phase === "running" || phase === "pending" || phase === "waiting_for_input";
          }).length;
        }

        let projectsCount: number | null = null;
        if (projectsResp.status === "fulfilled" && projectsResp.value.ok) {
          const data = await projectsResp.value.json();
          projectsCount = Array.isArray(data) ? data.length : (data.items?.length ?? null);
        }

        let templatesCount: number | null = null;
        if (templatesResp.status === "fulfilled" && templatesResp.value.ok) {
          const data = await templatesResp.value.json();
          templatesCount = Array.isArray(data) ? data.length : (data.items?.length ?? null);
        }

        let chainsCount: number | null = null;
        if (chainsResp.status === "fulfilled" && chainsResp.value.ok) {
          const data = await chainsResp.value.json();
          chainsCount = Array.isArray(data) ? data.length : (data.items?.length ?? null);
        }

        let chainrunsCount: number | null = null;
        if (chainrunsResp.status === "fulfilled" && chainrunsResp.value.ok) {
          const data = await chainrunsResp.value.json();
          chainrunsCount = Array.isArray(data) ? data.length : (data.items?.length ?? null);
        }

        let schedulesCount: number | null = null;
        if (schedulesResp.status === "fulfilled" && schedulesResp.value.ok) {
          const data = await schedulesResp.value.json();
          schedulesCount = Array.isArray(data) ? data.length : (data.items?.length ?? null);
        }

        if (!cancelled) {
          setCounts({ runs: runsCount, projects: projectsCount, templates: templatesCount, chains: chainsCount, chainruns: chainrunsCount, schedules: schedulesCount });
        }
      } catch {
        // silently ignore fetch errors for badge counts
      }
    };

    fetch();
    const interval = setInterval(() => {
      if (!cancelled) fetch();
    }, 10000);

    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, []);

  function isActive(item: NavItem): boolean {
    if (item.path === "/") {
      return location.pathname === "/" || location.pathname.startsWith("/run/") || location.pathname === "/new" || location.pathname.startsWith("/chainrun/");
    }
    if (item.path === "/chains") {
      return location.pathname === "/chains" || location.pathname.startsWith("/chains/");
    }
    return location.pathname === item.path || location.pathname.startsWith(item.path + "/");
  }

  function getCount(key: string): number | null {
    return counts[key as keyof Counts];
  }

  return (
    <div
      className={`flex flex-col h-screen border-r bg-background transition-all duration-200 shrink-0 ${
        collapsed ? "w-[50px]" : "w-[200px]"
      }`}
    >
      {/* Collapse toggle */}
      <div className="flex items-center border-b px-2 py-2">
        <button
          onClick={toggleCollapsed}
          className="flex items-center justify-center w-7 h-7 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted transition-colors text-xs"
          title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        >
          {collapsed ? "→" : "←"}
        </button>
        {!collapsed && (
          <span className="ml-2 text-xs font-semibold tracking-widest text-muted-foreground">
            NAV
          </span>
        )}
      </div>

      {/* Nav items */}
      <nav className="flex-1 py-2 flex flex-col gap-0.5 px-1.5">
        {NAV_ITEMS.map((item) => {
          const active = isActive(item);
          const count = getCount(item.countKey);
          const showBadge = count !== null && count > 0;

          return (
            <Link
              key={item.path}
              to={item.path}
              className={`flex items-center gap-2 px-2 py-1.5 rounded-md text-sm transition-colors ${
                active
                  ? "bg-accent text-accent-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground hover:bg-muted"
              } ${collapsed ? "justify-center" : ""}`}
              title={collapsed ? item.label : undefined}
            >
              <span className="w-4 text-center text-base leading-none shrink-0">{item.icon}</span>
              {!collapsed && (
                <>
                  <span className="flex-1 truncate">{item.label}</span>
                  {showBadge && (
                    <span
                      className={`text-xs font-mono px-1.5 py-0.5 rounded-full leading-none ${
                        item.countKey === "runs"
                          ? "bg-blue-500/20 text-blue-600 dark:text-blue-400"
                          : "bg-muted text-muted-foreground"
                      }`}
                    >
                      {count}
                    </span>
                  )}
                </>
              )}
              {collapsed && showBadge && (
                <span
                  className="absolute top-0 right-0 w-1.5 h-1.5 rounded-full bg-blue-500"
                  style={{ position: "relative", marginLeft: "-4px", marginTop: "-8px" }}
                />
              )}
            </Link>
          );
        })}
      </nav>

      {/* Footer: brand + theme toggle */}
      <div className={`border-t px-2 py-2 flex items-center ${collapsed ? "flex-col gap-2 justify-center" : "justify-between"}`}>
        <span className="text-xs text-muted-foreground tracking-widest font-semibold">
          {collapsed ? "UW" : "UNCWORKS"}
        </span>
        <button
          onClick={cycleMode}
          className="text-xs text-muted-foreground hover:text-foreground transition-colors px-1.5 py-0.5 rounded-md"
          title={`Theme: ${mode}`}
        >
          {collapsed ? "◐" : mode}
        </button>
      </div>
    </div>
  );
}
