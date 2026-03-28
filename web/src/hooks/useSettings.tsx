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
}

export const SETTINGS_DEFAULTS: AppSettings = {
  githubToken: "",
  namespace: "uncworks",
  kubeContext: "",
  portRangeStart: 50100,
  portRangeEnd: 50120,
  envOverrides: {},
};

// Features that require specific config
export interface ConfigStatus {
  hasLLMKey: boolean;
  hasGitHubToken: boolean;
  // Derived capability flags
  canUseAI: boolean;
  canAccessPrivateRepos: boolean;
  canCreatePRs: boolean;
}

export function deriveConfigStatus(s: AppSettings): ConfigStatus {
  const hasLLMKey = false;
  const hasGitHubToken = Boolean(s.githubToken?.trim());
  return {
    hasLLMKey,
    hasGitHubToken,
    canUseAI: hasLLMKey,
    canAccessPrivateRepos: hasGitHubToken,
    canCreatePRs: hasGitHubToken,
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
  reload: () => Promise<void>;
  save: (s: AppSettings) => Promise<void>;
}

const SettingsContext = createContext<SettingsContextValue>({
  settings: SETTINGS_DEFAULTS,
  configStatus: deriveConfigStatus(SETTINGS_DEFAULTS),
  loading: false,
  reload: async () => {},
  save: async () => {},
});

export function SettingsProvider({ children }: { children: ReactNode }) {
  const wails = isWails();
  const [settings, setSettings] = useState<AppSettings>(SETTINGS_DEFAULTS);
  const [loading, setLoading] = useState(true);

  const reload = useCallback(async () => {
    if (wails) {
      try {
        const s = await go().GetSettings();
        if (s) setSettings(s);
      } catch { /* keep defaults */ }
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
    <SettingsContext.Provider value={{ settings, configStatus, loading, reload, save }}>
      {children}
    </SettingsContext.Provider>
  );
}

export function useSettings() {
  return useContext(SettingsContext);
}
