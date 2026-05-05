// wails-env.ts — Detect and configure Wails desktop environment.
// Sets data-wails on <html> for CSS targeting and installs native-app UX fixes.

// ── Wails binding types ───────────────────────────────────────────────────────

export interface WailsAppSettings {
  githubToken: string;
  namespace: string;
  kubeContext: string;
  portRangeStart: number;
  portRangeEnd: number;
  envOverrides: Record<string, string>;
  litellmURL: string;
  githubAuthed: boolean;
  updateChannel: string;
  autoUpdateEnabled: boolean;
  defaultManageModel: string;
  defaultImplementModel: string;
  wizardComplete: boolean;
  apiserverURL: string;
  llmApiKey?: string;
  llmKeyConfigured?: boolean;
  showTrafficLights?: boolean;
  copilotModel?: string;
}

export interface WailsEnvVarInfo {
  key: string;
  system: string;
  override: string;
  desc: string;
}

export interface WailsServiceInfo {
  name: string;
  displayName: string;
  clusterPort: number;
  localPort: number;
  ready: boolean;
  forwarding: boolean;
}

export type WailsHealthStatus = "ok" | "degraded" | "down" | "unknown";

export interface WailsHealthComponent {
  name: string;
  label: string;
  status: WailsHealthStatus;
  message: string;
}

export interface WailsHealthReport {
  overall: WailsHealthStatus;
  components: WailsHealthComponent[];
}

export interface WailsLiteLLMCheckResult {
  ok: boolean;
  models: string[];
  error?: string;
}

export interface WailsUpdateInfo {
  localBuild: boolean;
  upToDate: boolean;
  currentVersion?: string;
  latestVersion?: string;
  releaseURL?: string;
  error?: string;
}

// Wails v2 converts Go JSON tags (snake_case) to camelCase in JS bindings.
export interface WailsDeviceFlowStart {
  deviceCode: string;
  userCode: string;
  verificationURI: string;
  expiresIn: number;
  interval: number;
}

export interface WailsDeviceFlowPollResult {
  done: boolean;
  token?: string;
}

export interface WailsApp {
  GetSettings(): Promise<WailsAppSettings>;
  SaveSettings(s: WailsAppSettings): Promise<void>;
  GetEnvVars(): Promise<WailsEnvVarInfo[]>;
  GetKubeContexts(): Promise<string[]>;
  AutodetectNamespace(kubeContext: string): Promise<string>;
  ListServices(): Promise<WailsServiceInfo[]>;
  RestartService(name: string): Promise<void>;
  StartPortForward(name: string, localPort: number): Promise<void>;
  StopPortForward(name: string): Promise<void>;
  OpenURL(rawURL: string): Promise<void>;
  OpenService(name: string): Promise<void>;
  HealthCheck(): Promise<WailsHealthReport>;
  CheckLiteLLM(url: string): Promise<WailsLiteLLMCheckResult>;
  CheckForUpdate(): Promise<WailsUpdateInfo>;
  StartGitHubDeviceFlow(): Promise<WailsDeviceFlowStart>;
  PollGitHubDeviceFlow(deviceCode: string): Promise<WailsDeviceFlowPollResult>;
  SaveGitHubToken(token: string): Promise<void>;
  GetGitHubUser(): Promise<string>;
  DisconnectGitHub(): Promise<void>;
  OpenLogInConsole(): Promise<void>;
  LogPath(): Promise<string>;
}

declare global {
  interface Window {
    go?: { main?: { App?: WailsApp } };
    runtime?: { EventsOn(event: string, callback: (...args: unknown[]) => void): void };
  }
}

// ── Environment helpers ───────────────────────────────────────────────────────

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
