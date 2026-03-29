// SetupWizard.tsx — Multi-step first-run setup wizard.
// Steps: 1) Cluster  2) GitHub OAuth  3) Models
import { useState, useEffect, useCallback } from "react";
import { useSettings } from "../hooks/useSettings";

/* eslint-disable @typescript-eslint/no-explicit-any */
const go = () => (window as any).go?.main?.App;

type Step = "cluster" | "github" | "models";
const STEPS: Step[] = ["cluster", "github", "models"];
const STEP_LABELS: Record<Step, string> = {
  cluster: "Cluster",
  github: "GitHub",
  models: "Models",
};

interface Props {
  onClose: () => void;
}

export default function SetupWizardModal({ onClose }: Props) {
  const [step, setStep] = useState<Step>("cluster");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="relative w-full max-w-lg bg-background border rounded-xl shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="px-6 pt-6 pb-4 border-b">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-base font-semibold">Setup UNCWORKS</h2>
            <button
              onClick={onClose}
              className="text-muted-foreground hover:text-foreground transition-colors text-sm"
            >
              ✕
            </button>
          </div>
          {/* Progress indicator */}
          <div className="flex gap-2">
            {STEPS.map((s, i) => (
              <div key={s} className="flex items-center gap-2 flex-1">
                <div
                  className={`flex items-center justify-center w-6 h-6 rounded-full text-xs font-medium shrink-0 transition-colors ${
                    s === step
                      ? "bg-accent text-accent-foreground"
                      : STEPS.indexOf(step) > i
                      ? "bg-green-500/20 text-green-600 dark:text-green-400"
                      : "bg-muted text-muted-foreground"
                  }`}
                >
                  {STEPS.indexOf(step) > i ? "✓" : i + 1}
                </div>
                <span
                  className={`text-xs transition-colors ${
                    s === step ? "text-foreground font-medium" : "text-muted-foreground"
                  }`}
                >
                  {STEP_LABELS[s]}
                </span>
                {i < STEPS.length - 1 && (
                  <div className={`flex-1 h-px mx-1 transition-colors ${STEPS.indexOf(step) > i ? "bg-green-500/40" : "bg-border"}`} />
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Body */}
        <div className="px-6 py-5 min-h-[220px]">
          {step === "cluster" && <ClusterStep onNext={() => setStep("github")} />}
          {step === "github" && <GitHubStep onNext={() => setStep("models")} onSkip={() => setStep("models")} />}
          {step === "models" && <ModelsStep onFinish={onClose} />}
        </div>
      </div>
    </div>
  );
}

// ── Step 1: Cluster ────────────────────────────────────────────────────────────

function ClusterStep({ onNext }: { onNext: () => void }) {
  const { settings, save } = useSettings();
  const [contexts, setContexts] = useState<string[]>([]);
  const [selectedCtx, setSelectedCtx] = useState(settings.kubeContext || "");
  const [namespace, setNamespace] = useState(settings.namespace || "uncworks");
  const [detecting, setDetecting] = useState(false);
  const [detected, setDetected] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    go().GetKubeContexts().then((ctxs: string[]) => {
      setContexts(ctxs ?? []);
      if (!selectedCtx && ctxs?.length > 0) setSelectedCtx(ctxs[0]);
    }).catch(() => {}).finally(() => setLoading(false));
  }, []);

  const detectNamespace = useCallback(async (ctx: string) => {
    if (!ctx) return;
    setDetecting(true);
    setDetected(null);
    try {
      const ns = await go().AutodetectNamespace(ctx);
      if (ns) {
        setDetected(ns);
        setNamespace(ns);
      }
    } catch { /* ignore */ }
    finally { setDetecting(false); }
  }, []);

  useEffect(() => {
    if (selectedCtx) detectNamespace(selectedCtx);
  }, [selectedCtx, detectNamespace]);

  async function onContinue() {
    await save({ ...settings, kubeContext: selectedCtx, namespace });
    onNext();
  }

  return (
    <div>
      <p className="text-sm text-muted-foreground mb-5">
        Choose the Kubernetes cluster and namespace where UNCWORKS is deployed.
      </p>

      <div className="mb-4">
        <label className="text-xs font-medium mb-1 block">Cluster context</label>
        {loading ? (
          <p className="text-xs text-muted-foreground">Loading contexts…</p>
        ) : contexts.length === 0 ? (
          <p className="text-xs text-muted-foreground">No kubecontexts found — is kubectl configured?</p>
        ) : (
          <select
            value={selectedCtx}
            onChange={e => setSelectedCtx(e.target.value)}
            className="w-full px-2.5 py-1.5 rounded-md border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-ring"
          >
            {contexts.map(c => <option key={c} value={c}>{c}</option>)}
          </select>
        )}
      </div>

      <div className="mb-6">
        <label className="text-xs font-medium mb-1 block">
          Namespace
          {detecting && <span className="ml-2 text-muted-foreground font-normal">detecting…</span>}
          {detected && !detecting && (
            <span className="ml-2 px-1.5 py-0.5 rounded-full text-xs bg-green-500/15 text-green-600 dark:text-green-400">
              detected
            </span>
          )}
        </label>
        <input
          type="text"
          value={namespace}
          onChange={e => setNamespace(e.target.value)}
          placeholder="uncworks"
          className="w-full px-2.5 py-1.5 rounded-md border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-ring"
        />
      </div>

      <div className="flex justify-end">
        <button
          onClick={onContinue}
          disabled={!selectedCtx}
          className="px-4 py-1.5 rounded-md bg-accent text-accent-foreground text-sm hover:opacity-90 disabled:opacity-40 transition-opacity"
        >
          Continue →
        </button>
      </div>
    </div>
  );
}

// ── Step 2: GitHub ─────────────────────────────────────────────────────────────

interface DeviceFlow {
  userCode: string;
  verificationURI: string;  // Go field: VerificationURI (json: verification_uri)
  deviceCode: string;
}

function GitHubStep({ onNext, onSkip }: { onNext: () => void; onSkip: () => void }) {
  const { settings, reload } = useSettings();
  const [flow, setFlow] = useState<DeviceFlow | null>(null);
  const [polling, setPolling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [authed, setAuthed] = useState(false);

  async function startFlow() {
    setError(null);
    try {
      const result = await go().StartGitHubDeviceFlow();
      setFlow(result);
      setPolling(true);
    } catch (e: any) {
      setError(String(e));
    }
  }

  useEffect(() => {
    if (!polling || !flow) return;
    let stopped = false;

    async function poll() {
      while (!stopped && flow) {
        await new Promise(r => setTimeout(r, 5000));
        try {
          const result = await go().PollGitHubDeviceFlow(flow.deviceCode);
          if (result?.done) {
            setAuthed(true);
            setPolling(false);
            await reload();
            return;
          }
          // result.done === false means still pending — keep polling
        } catch { /* keep polling */ }
      }
    }

    poll();
    return () => { stopped = true; };
  }, [polling, flow, reload]);

  if (authed) {
    return (
      <div>
        <div className="flex flex-col items-center py-6 gap-3">
          <div className="text-3xl">✓</div>
          <p className="text-sm font-medium">GitHub connected</p>
          <p className="text-xs text-muted-foreground">Your account has been authenticated.</p>
        </div>
        <div className="flex justify-end">
          <button
            onClick={onNext}
            className="px-4 py-1.5 rounded-md bg-accent text-accent-foreground text-sm hover:opacity-90 transition-opacity"
          >
            Continue →
          </button>
        </div>
      </div>
    );
  }

  if (flow) {
    return (
      <div>
        <p className="text-sm text-muted-foreground mb-4">
          Open the URL below and enter the code to authenticate:
        </p>
        <div className="rounded-md border bg-muted/30 p-4 mb-4 text-center">
          <p className="text-2xl font-mono font-bold tracking-widest mb-3">{flow.userCode}</p>
          <button
            onClick={() => go().OpenURL(flow.verificationURI)}
            className="text-sm text-accent-foreground hover:underline"
          >
            {flow.verificationURI}
          </button>
        </div>
        {polling && (
          <p className="text-xs text-muted-foreground text-center animate-pulse mb-4">
            Waiting for authentication…
          </p>
        )}
        <div className="flex justify-between">
          <button onClick={() => { setFlow(null); setPolling(false); }} className="text-xs text-muted-foreground hover:text-foreground transition-colors">
            Cancel
          </button>
          <button onClick={onSkip} className="text-xs text-muted-foreground hover:text-foreground transition-colors">
            Skip →
          </button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <p className="text-sm text-muted-foreground mb-5">
        Connect GitHub to enable private repo access and pull request creation.
      </p>
      {error && (
        <div className="mb-4 text-xs text-destructive px-3 py-2 rounded-md bg-destructive/10">{error}</div>
      )}

      {settings.githubToken?.trim() && (
        <div className="mb-4 px-3 py-2 rounded-md bg-green-500/10 text-green-600 dark:text-green-400 text-xs">
          Personal access token already configured.
        </div>
      )}

      <div className="flex gap-2 justify-end">
        <button
          onClick={onSkip}
          className="px-3 py-1.5 rounded-md border text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          Skip
        </button>
        <button
          onClick={startFlow}
          className="px-4 py-1.5 rounded-md bg-accent text-accent-foreground text-sm hover:opacity-90 transition-opacity"
        >
          Connect GitHub
        </button>
      </div>
    </div>
  );
}

// ── Step 3: Models ─────────────────────────────────────────────────────────────

interface ProviderPreset {
  label: string;
  url: string;
  hint: string;
}

const PROVIDER_PRESETS: ProviderPreset[] = [
  { label: "Cluster (built-in)",   url: "http://litellm:4000",         hint: "Uses the LiteLLM proxy deployed in your cluster." },
  { label: "OpenRouter",           url: "https://openrouter.ai/api/v1", hint: "Route to any model via openrouter.ai. Set your API key in the cluster." },
  { label: "OpenAI",               url: "https://api.openai.com/v1",    hint: "Connect directly to OpenAI. Set your API key in the cluster." },
  { label: "Custom",               url: "",                             hint: "Any OpenAI-compatible endpoint." },
];

function ModelsStep({ onFinish }: { onFinish: () => void }) {
  const { settings, save } = useSettings();

  // Pick the initial preset based on current saved URL.
  function initialPreset() {
    const saved = settings.litellmURL || "";
    const match = PROVIDER_PRESETS.find(p => p.url && p.url === saved);
    return match?.label ?? (saved ? "Custom" : "Cluster (built-in)");
  }

  const [preset, setPreset] = useState(initialPreset);
  const [url, setUrl] = useState(settings.litellmURL || "http://litellm:4000");
  const [checking, setChecking] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; models: string[]; error?: string } | null>(null);

  function onPresetChange(label: string) {
    setPreset(label);
    setResult(null);
    const p = PROVIDER_PRESETS.find(p => p.label === label);
    if (p && p.url) setUrl(p.url);
  }

  async function check() {
    setChecking(true);
    setResult(null);
    try {
      const r = await go().CheckLiteLLM(url);
      setResult(r);
    } catch (e: any) {
      setResult({ ok: false, models: [], error: String(e) });
    } finally {
      setChecking(false);
    }
  }

  async function finish() {
    await save({ ...settings, litellmURL: url, wizardComplete: true });
    onFinish();
  }

  const currentPreset = PROVIDER_PRESETS.find(p => p.label === preset);
  const isCluster = preset === "Cluster (built-in)";

  return (
    <div>
      <p className="text-sm text-muted-foreground mb-5">
        Choose how agents connect to AI models. All providers use an OpenAI-compatible endpoint.
      </p>

      <div className="mb-4">
        <label className="text-xs font-medium mb-1 block">Provider</label>
        <select
          value={preset}
          onChange={e => onPresetChange(e.target.value)}
          className="w-full px-2.5 py-1.5 rounded-md border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-ring"
        >
          {PROVIDER_PRESETS.map(p => (
            <option key={p.label} value={p.label}>{p.label}</option>
          ))}
        </select>
        {currentPreset && (
          <p className="mt-1 text-xs text-muted-foreground">{currentPreset.hint}</p>
        )}
      </div>

      <div className="mb-4">
        <label className="text-xs font-medium mb-1 block">Endpoint URL</label>
        <div className="flex gap-2">
          <input
            type="text"
            value={url}
            onChange={e => { setUrl(e.target.value); setResult(null); }}
            placeholder="https://..."
            className="flex-1 px-2.5 py-1.5 rounded-md border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-ring"
          />
          <button
            onClick={check}
            disabled={checking || !url}
            className="px-3 py-1.5 rounded-md border text-xs hover:bg-accent transition-colors disabled:opacity-50 whitespace-nowrap"
          >
            {checking ? "Testing…" : "Test"}
          </button>
        </div>
        {result && (
          <div className={`mt-2 text-xs ${result.ok ? "text-green-600 dark:text-green-400" : "text-destructive"}`}>
            {result.ok
              ? `Connected — ${result.models.length} model(s) available`
              : result.error || "Connection failed"}
          </div>
        )}
        {result && !result.ok && isCluster && (
          <p className="mt-1 text-xs text-muted-foreground">
            Make sure your cluster is running and the port-forward is active.
          </p>
        )}
      </div>

      <div className="flex justify-between items-center">
        <button
          onClick={finish}
          className="text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          Skip for now
        </button>
        <button
          onClick={finish}
          className="px-4 py-1.5 rounded-md bg-accent text-accent-foreground text-sm hover:opacity-90 transition-opacity"
        >
          Finish
        </button>
      </div>
    </div>
  );
}
