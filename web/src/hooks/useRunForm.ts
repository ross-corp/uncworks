import { useState, useCallback } from "react";
import type { OrchestrationMode } from "../types/agent-run";

interface RepoEntry {
  url: string;
  branch: string;
}

export interface RunFormState {
  prompt: string;
  repos: RepoEntry[];
  mode: "prompt" | "spec";
  specContent: string;
  modelTier: string;
  ttlMinutes: number;
  orchestrationMode: OrchestrationMode;
  implementModelTier: string;
  projectRef: string;
  specRef: string;
  customLabelMode: boolean;
  project: string;
  feature: string;
  tags: string;
}

const DEFAULT_FORM: RunFormState = {
  prompt: "",
  repos: [{ url: "https://github.com/roshbhatia/neph.nvim", branch: "main" }],
  mode: "prompt",
  specContent: "",
  modelTier: "default",
  ttlMinutes: 15,
  orchestrationMode: "single",
  implementModelTier: "",
  projectRef: "",
  specRef: "",
  customLabelMode: false,
  project: "",
  feature: "",
  tags: "",
};

type SetField = {
  [K in keyof RunFormState]: (value: RunFormState[K]) => void;
};

export interface UseRunFormReturn {
  form: RunFormState;
  set: SetField;
  reset: () => void;
}

export function useRunForm(): UseRunFormReturn {
  const [form, setForm] = useState<RunFormState>({ ...DEFAULT_FORM });

  const set = {} as SetField;
  (Object.keys(DEFAULT_FORM) as (keyof RunFormState)[]).forEach((key) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (set as any)[key] = (value: unknown) =>
      setForm((prev) => ({ ...prev, [key]: value }));
  });

  const reset = useCallback(() => setForm({ ...DEFAULT_FORM }), []);

  return { form, set, reset };
}
