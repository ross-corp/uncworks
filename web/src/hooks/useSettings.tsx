// useSettings.tsx — Global settings context. Loads once from Wails (desktop) or
// localStorage (web). Derives ConfigStatus so any view can gate features on config.
import { createContext, useContext, useEffect, useState, useCallback, ReactNode } from "react";
import { isWails } from "../lib/wails-env";

/* eslint-disable @typescript-eslint/no-explicit-any */
const go = () => (window as any).go?.main?.App;

export interface AppSettings {
  githubToken: string;
  namespace: string;
  kubeContext: string;
  portRangeStart: number;
  portRangeEnd: number;
  envOverrides: Record<string, string>;
  // New fields added in settings-wizard
  litellmURL: string;
  githubAuthed: boolean;
  updateChannel: string;
  autoUpdateEnabled: boolean;
  defaultManageModel: string;
  defaultImplementModel: string;
  wizardComplete: boolean;
  apiserverURL: string;
  // LLM API key for external providers (e.g. OpenRouter).
  llmApiKey?: string;
  // Computed by the Go backend: true when llmApiKey is non-empty.
  llmKeyConfigured?: boolean;
  // Controls traffic-light visibility. Change takes effect on next launch.
  showTrafficLights?: boolean;
  // Model used by the copilot panel. Empty string means LiteLLM's default model.
  copilotModel?: string;
}

export const SETTINGS_DEFAULTS: AppSettings = {
  githubToken: "",
  namespace: "uncworks",
  kubeContext: "",
  portRangeStart: 50100,
  portRangeEnd: 50120,
  envOverrides: {},
  litellmURL: "http://litellm:4000",
  githubAuthed: false,
  updateChannel: "stable",
  autoUpdateEnabled: false,
  defaultManageModel: "",
  defaultImplementModel: "",
  wizardComplete: false,
  apiserverURL: "http://localhost:50055",
};

// Features that require specific config
export interface ConfigStatus {
  hasLLMKey: boolean;
  hasGitHubToken: boolean;
  hasGitHubOAuth: boolean;
  wizardComplete: boolean;
  // Derived capability flags
  canUseAI: boolean;
  canAccessPrivateRepos: boolean;
  canCreatePRs: boolean;
}

function isClusterLiteLLM(url: string | undefined): boolean {
  if (!url) return true;
  return url === "http://litellm:4000" || url.startsWith("http://localhost:");
}

export function deriveConfigStatus(s: AppSettings): ConfigStatus {
  // Cluster LiteLLM (port-forwarded to localhost or in-cluster) needs no external key.
  const hasLLMKey =
    Boolean(s.llmKeyConfigured) ||
    Boolean(s.llmApiKey?.trim()) ||
    isClusterLiteLLM(s.litellmURL);
  const hasGitHubToken = Boolean(s.githubToken?.trim());
  const hasGitHubOAuth = Boolean(s.githubAuthed);
  return {
    hasLLMKey,
    hasGitHubToken,
    hasGitHubOAuth,
    wizardComplete: Boolean(s.wizardComplete),
    canUseAI: hasLLMKey,
    canAccessPrivateRepos: hasGitHubToken || hasGitHubOAuth,
    canCreatePRs: hasGitHubToken || hasGitHubOAuth,
  };
}

const LS_KEY = "uncworks-settings";

function loadFromStorage(): AppSettings {
  try {
    const raw = localStorage.getItem(LS_KEY);
    if (raw) return { ...SETTINGS_DEFAULTS, ...JSON.parse(raw) };
  } catch { /* ignore */ }
  return { ...SETTINGS_DEFAULTS };
}

interface SettingsContextValue {
  settings: AppSettings;
  configStatus: ConfigStatus;
  loading: boolean;
  error: Error | null;
  reload: () => Promise<void>;
  save: (s: AppSettings) => Promise<void>;
}

const SettingsContext = createContext<SettingsContextValue>({
  settings: SETTINGS_DEFAULTS,
  configStatus: deriveConfigStatus(SETTINGS_DEFAULTS),
  loading: false,
  error: null,
  reload: async () => {},
  save: async () => {},
});

export function SettingsProvider({ children }: { children: ReactNode }) {
  const wails = isWails();
  const [settings, setSettings] = useState<AppSettings>(SETTINGS_DEFAULTS);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const reload = useCallback(async () => {
    setError(null);
    if (wails) {
      try {
        const s = await go().GetSettings();
        if (s) setSettings(s);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
      }
    } else {
      setSettings(loadFromStorage());
    }
    setLoading(false);
  }, [wails]);

  useEffect(() => { reload(); }, [reload]);

  const save = useCallback(async (s: AppSettings) => {
    if (wails) {
      await go().SaveSettings(s);
    } else {
      localStorage.setItem(LS_KEY, JSON.stringify(s));
    }
    setSettings(s);
  }, [wails]);

  const configStatus = deriveConfigStatus(settings);

  return (
    <SettingsContext.Provider value={{ settings, configStatus, loading, error, reload, save }}>
      {children}
    </SettingsContext.Provider>
  );
}

export function useSettings() {
  return useContext(SettingsContext);
}
