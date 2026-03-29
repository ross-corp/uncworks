// GlobalNav.tsx — Resizable sidebar with ROSS CORP logo, nav items, and config status indicator.
import { useState, useRef, useCallback, useEffect } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { isWails } from "../lib/wails-env";
import {
  Play,
  FolderOpen,
  LayoutTemplate,
  Link2,
  Clock,
  Settings,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import { useSettings } from "../hooks/useSettings";

// ROSS CORP org avatar — same image works on light and dark backgrounds
const LOGO_URL = "https://avatars.githubusercontent.com/u/172242530?s=64";

interface NavItem {
  label: string;
  path: string;
  Icon: React.ComponentType<{ size?: number; strokeWidth?: number }>;
  countKey: string;
}

interface NavGroup {
  label: string;
  items: NavItem[];
}

const NAV_GROUPS: NavGroup[] = [
  {
    label: "Activity",
    items: [
      { label: "Runs", path: "/", Icon: Play, countKey: "runs" },
    ],
  },
  {
    label: "Library",
    items: [
      { label: "Projects",  path: "/projects",  Icon: FolderOpen,     countKey: "projects"  },
      { label: "Templates", path: "/templates", Icon: LayoutTemplate, countKey: "templates" },
    ],
  },
  {
    label: "Automation",
    items: [
      { label: "Chains",    path: "/chains",    Icon: Link2,  countKey: "chains"    },
      { label: "Schedules", path: "/schedules", Icon: Clock,  countKey: "schedules" },
    ],
  },
];

interface Counts {
  runs: number | null;
  projects: number | null;
  templates: number | null;
  chains: number | null;
  chainruns: number | null;
  schedules: number | null;
}

const MIN_WIDTH = 50;
const COLLAPSED_WIDTH = 50;
const DEFAULT_WIDTH = 200;
const MAX_WIDTH = 320;

function loadWidth(): number {
  const v = localStorage.getItem("nav-width");
  return v ? Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, Number(v))) : DEFAULT_WIDTH;
}

export default function GlobalNav() {
  const location = useLocation();
  const navigate = useNavigate();
  const { configStatus } = useSettings();
  const wails = isWails();

  const [width, setWidth] = useState<number>(loadWidth);
  const [collapsed, setCollapsed] = useState<boolean>(() => loadWidth() <= COLLAPSED_WIDTH + 20);
  const [counts, setCounts] = useState<Counts>({
    runs: null, projects: null, templates: null,
    chains: null, chainruns: null, schedules: null,
  });

  const dragging = useRef(false);
  const startX = useRef(0);
  const startW = useRef(0);
  const navRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    localStorage.setItem("nav-width", String(width));
  }, [width]);

  // Live counts from API
  useEffect(() => {
    let cancelled = false;
    async function poll() {
      try {
        const resp = await fetch("/api/v1/counts");
        if (!cancelled && resp.ok) {
          const data = await resp.json();
          setCounts({
            runs: data.activeRuns ?? null,
            projects: data.projects ?? null,
            templates: data.templates ?? null,
            chains: data.chains ?? null,
            chainruns: data.chainruns ?? null,
            schedules: data.schedules ?? null,
          });
        }
      } catch { /* ignore — API may not be reachable */ }
    }
    poll();
    const t = setInterval(poll, 10000);
    return () => { cancelled = true; clearInterval(t); };
  }, []);

  // Settings navigation event (Cmd+, or macOS Preferences menu)
  useEffect(() => {
    function onOpenSettings() { navigate("/settings"); }
    window.addEventListener("uncworks:open-settings", onOpenSettings);
    return () => window.removeEventListener("uncworks:open-settings", onOpenSettings);
  }, [navigate]);

  // Drag-to-resize
  const onMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    dragging.current = true;
    startX.current = e.clientX;
    startW.current = width;

    function onMove(ev: MouseEvent) {
      if (!dragging.current) return;
      const next = Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, startW.current + ev.clientX - startX.current));
      setWidth(next);
      setCollapsed(next <= COLLAPSED_WIDTH + 20);
    }
    function onUp() {
      dragging.current = false;
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    }
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  }, [width]);

  function toggleCollapsed() {
    if (collapsed) {
      setWidth(DEFAULT_WIDTH);
      setCollapsed(false);
    } else {
      setWidth(COLLAPSED_WIDTH);
      setCollapsed(true);
    }
  }

  function isActive(item: NavItem): boolean {
    if (item.path === "/") {
      return (
        location.pathname === "/" ||
        location.pathname.startsWith("/run/") ||
        location.pathname === "/new"
      );
    }
    if (item.path === "/chains") {
      return (
        location.pathname === "/chains" ||
        location.pathname.startsWith("/chains/") ||
        location.pathname === "/chainruns" ||
        location.pathname.startsWith("/chainrun/")
      );
    }
    return location.pathname === item.path || location.pathname.startsWith(item.path + "/");
  }

  const isSettings = location.pathname === "/settings";

  // Show a dot when config is incomplete (guides user to settings)
  const configIncomplete = !configStatus.hasGitHubToken;

  return (
    <div
      ref={navRef}
      className="relative flex flex-col h-full border-r bg-background shrink-0 select-none"
      style={{ width }}
    >
      {/* Logo + collapse toggle */}
      <div className={`flex items-center px-2 py-2.5 gap-2 ${collapsed ? "justify-center" : ""}${wails ? " pt-7" : ""}`}>
        {/* Logo — always visible */}
        <button
          onClick={toggleCollapsed}
          className="shrink-0 w-7 h-7 rounded-full overflow-hidden focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        >
          <img
            src={LOGO_URL}
            alt="ROSS CORP"
            className="w-full h-full object-cover"
            draggable={false}
          />
        </button>

        {/* Expand / collapse chevron, only when not collapsed */}
        {!collapsed && (
          <>
            <span className="flex-1 text-xs font-semibold tracking-widest text-muted-foreground truncate">
              UNCWORKS
            </span>
            <button
              onClick={toggleCollapsed}
              className="shrink-0 w-5 h-5 flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
              aria-label="Collapse sidebar"
              title="Collapse sidebar"
            >
              <ChevronLeft size={14} strokeWidth={2} />
            </button>
          </>
        )}
      </div>

      {/* Expand chevron when collapsed */}
      {collapsed && (
        <div className="flex justify-center px-2 pb-1">
          <button
            onClick={toggleCollapsed}
            className="w-5 h-5 flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
            aria-label="Expand sidebar"
            title="Expand sidebar"
          >
            <ChevronRight size={14} strokeWidth={2} />
          </button>
        </div>
      )}

      {/* Nav items */}
      <nav className="flex-1 flex flex-col gap-0.5 px-1.5 pt-1 overflow-hidden">
        {NAV_GROUPS.map((group, groupIdx) => (
          <div key={group.label} className={groupIdx > 0 ? "mt-2" : ""}>
            {!collapsed && (
              <span className="block px-2 pb-0.5 text-[10px] font-semibold tracking-widest uppercase text-muted-foreground/50">
                {group.label}
              </span>
            )}
            {group.items.map((item) => {
              const active = isActive(item);
              const count = counts[item.countKey as keyof Counts];
              const showBadge = count !== null && count > 0;
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`flex items-center gap-2.5 px-2 py-2 rounded-md text-sm transition-colors ${
                    active
                      ? "bg-accent text-accent-foreground"
                      : "text-muted-foreground hover:text-foreground hover:bg-muted"
                  } ${collapsed ? "justify-center" : ""}`}
                  title={collapsed ? item.label : undefined}
                >
                  <item.Icon size={15} strokeWidth={active ? 2.5 : 2} />
                  {!collapsed && (
                    <>
                      <span className="flex-1 truncate">{item.label}</span>
                      {showBadge && (
                        <span className="text-xs font-mono tabular-nums px-1.5 py-0.5 rounded-full leading-none bg-blue-500/15 text-blue-600 dark:text-blue-400">
                          {count}
                        </span>
                      )}
                    </>
                  )}
                </Link>
              );
            })}
          </div>
        ))}
      </nav>

      {/* Settings link — with optional config-incomplete dot */}
      <div className="px-1.5 pb-3 pt-2 border-t">
        <Link
          to="/settings"
          className={`flex items-center gap-2.5 px-2 py-2 rounded-md text-sm transition-colors relative ${
            isSettings
              ? "bg-accent text-accent-foreground"
              : "text-muted-foreground hover:text-foreground hover:bg-muted"
          } ${collapsed ? "justify-center" : ""}`}
          title={collapsed ? "Settings" : undefined}
        >
          <div className="relative shrink-0">
            <Settings size={15} strokeWidth={isSettings ? 2.5 : 2} />
            {configIncomplete && (
              <span
                className="absolute -top-0.5 -right-0.5 w-1.5 h-1.5 rounded-full bg-amber-500"
                title="Configuration incomplete"
              />
            )}
          </div>
          {!collapsed && <span className="flex-1 truncate">Settings</span>}
        </Link>
      </div>

      {/* Resize handle */}
      <div
        onMouseDown={onMouseDown}
        className="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-accent/30 active:bg-accent/60 transition-colors z-10"
        style={{ touchAction: "none" }}
      />
    </div>
  );
}
