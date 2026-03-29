// SettingsView.tsx — App settings: credentials, cluster, appearance, services, advanced.
import { useEffect, useState, useCallback } from "react";
import { isWails } from "../lib/wails-env";
import { useThemeNew, type ColorMode } from "../hooks/useThemeNew";
import { useSettings, type AppSettings } from "../hooks/useSettings";
import SetupWizardModal, { GitHubAuthModal } from "../components/SetupWizard";

/* eslint-disable @typescript-eslint/no-explicit-any */
const go = () => (window as any).go?.main?.App;

interface EnvVarInfo {
  key: string;
  system: string;
  override: string;
  desc: string;
}

interface ServiceInfo {
  name: string;
  displayName: string;
  clusterPort: number;
  localPort: number;
  ready: boolean;
  forwarding: boolean;
}

const MODES: { value: ColorMode; label: string }[] = [
  { value: "system", label: "System" },
  { value: "light",  label: "Light"  },
  { value: "dark",   label: "Dark"   },
];

const FONT_KEY = "unc-font-size";
const FONT_DEFAULT = 13;

export default function SettingsView() {
  const wails = isWails();
  const { mode, setMode } = useThemeNew();
  const { settings: globalSettings, configStatus, reload: reloadGlobal, save: saveGlobal } = useSettings();

  const [local, setLocal] = useState<AppSettings>(globalSettings);
  const [dirty, setDirty] = useState(false);
  const [services, setServices] = useState<ServiceInfo[]>([]);
  const [envVars, setEnvVars] = useState<EnvVarInfo[]>([]);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [restarting, setRestarting] = useState<Record<string, boolean>>({});
  const [error, setError] = useState<string | null>(null);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [showWizard, setShowWizard] = useState(false);
  const [showGitHubModal, setShowGitHubModal] = useState(false);

  // LiteLLM check state
  const [litellmChecking, setLitellmChecking] = useState(false);
  const [litellmResult, setLitellmResult] = useState<{ ok: boolean; models: string[]; error?: string } | null>(null);

  // GitHub auth state
  const [ghUser, setGhUser] = useState<string | null>(null);
  const [ghLoading, setGhLoading] = useState(false);
  const [ghTestResult, setGhTestResult] = useState<{ ok: boolean; login?: string } | null>(null);

  // Auto-update state
  const [updateInfo, setUpdateInfo] = useState<{ localBuild?: boolean; upToDate?: boolean; currentVersion?: string; latestVersion?: string; releaseURL?: string } | null>(null);
  const [updateChecking, setUpdateChecking] = useState(false);

  const [fontSize, setFontSizeState] = useState<number>(() => {
    const v = localStorage.getItem(FONT_KEY);
    return v ? Number(v) : FONT_DEFAULT;
  });

  // Sync local draft when global settings load/change
  useEffect(() => {
    setLocal(globalSettings);
    setDirty(false);
  }, [globalSettings]);

  const loadOperational = useCallback(async () => {
    if (!wails) return;
    try {
      const [svcs, evars] = await Promise.all([
        go().ListServices(),
        go().GetEnvVars(),
      ]);
      setServices(svcs ?? []);
      setEnvVars(evars ?? []);
    } catch (e: any) {
      setError(String(e));
    }
  }, [wails]);

  const loadGitHubUser = useCallback(async () => {
    if (!wails) return;
    try {
      const user = await go().GetGitHubUser();
      setGhUser(user || null);
    } catch { setGhUser(null); }
  }, [wails]);

  useEffect(() => { loadOperational(); loadGitHubUser(); }, [loadOperational, loadGitHubUser]);

  async function checkLiteLLM() {
    if (!wails) return;
    setLitellmChecking(true);
    setLitellmResult(null);
    try {
      const result = await go().CheckLiteLLM(local.litellmURL || "");
      setLitellmResult(result);
    } catch (e: any) {
      setLitellmResult({ ok: false, models: [], error: String(e) });
    } finally {
      setLitellmChecking(false);
    }
  }

  async function testGitHub() {
    if (!wails) return;
    setGhLoading(true);
    setGhTestResult(null);
    try {
      const login = await go().GetGitHubUser();
      setGhTestResult({ ok: !!login, login: login || undefined });
      if (login) setGhUser(login);
    } catch { setGhTestResult({ ok: false }); }
    finally { setGhLoading(false); }
  }

  async function disconnectGitHub() {
    if (!wails) return;
    setGhLoading(true);
    try {
      await go().DisconnectGitHub();
      await reloadGlobal();
      setGhUser(null);
    } catch (e: any) { setError(String(e)); }
    finally { setGhLoading(false); }
  }

  async function checkUpdate() {
    if (!wails) return;
    setUpdateChecking(true);
    try {
      const info = await go().CheckForUpdate();
      setUpdateInfo(info);
    } catch (e: any) { setError(String(e)); }
    finally { setUpdateChecking(false); }
  }

  function set<K extends keyof AppSettings>(key: K, value: AppSettings[K]) {
    setLocal(s => ({ ...s, [key]: value }));
    setDirty(true);
  }

  async function save() {
    setSaving(true);
    setError(null);
    try {
      await saveGlobal(local);
      await reloadGlobal();
      setDirty(false);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (e: any) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  }

  async function restart(name: string) {
    setRestarting(r => ({ ...r, [name]: true }));
    try {
      await go().RestartService(name);
      setTimeout(loadOperational, 3000);
    } catch (e: any) { setError(String(e)); }
    finally { setRestarting(r => ({ ...r, [name]: false })); }
  }

  function changeFont(px: number) {
    const clamped = Math.max(10, Math.min(24, px));
    setFontSizeState(clamped);
    document.documentElement.style.fontSize = `${clamped}px`;
    localStorage.setItem(FONT_KEY, String(clamped));
  }

  // Config notices — show only what actually breaks without each credential
  const notices: { key: string; message: string; anchor: string }[] = [];
  if (!configStatus.hasGitHubToken && !configStatus.hasGitHubOAuth)
    notices.push({ key: "gh", message: "Private repo cloning and PR creation require GitHub authentication.", anchor: "section-github" });

  return (
    // Outer shell fills the flex-col <main> from Layout; inner div scrolls
    <div className="flex h-full flex-col">
      {showWizard && <SetupWizardModal onClose={() => { setShowWizard(false); reloadGlobal(); loadGitHubUser(); }} />}
      {showGitHubModal && <GitHubAuthModal onClose={() => { setShowGitHubModal(false); reloadGlobal(); loadGitHubUser(); }} />}
      <div className="flex-1 min-h-0 overflow-y-auto overscroll-none">
      <div className="px-8 py-8 max-w-2xl">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-base font-semibold tracking-tight">Settings</h1>
          {wails && (
            <button
              onClick={() => setShowWizard(true)}
              className="text-xs text-muted-foreground hover:text-foreground border rounded-md px-2.5 py-1 transition-colors"
            >
              Re-run setup wizard
            </button>
          )}
        </div>

        {/* Config notices */}
        {notices.length > 0 && (
          <div className="mb-6 rounded-md border border-amber-500/30 bg-amber-500/5 divide-y divide-amber-500/20">
            {notices.map(n => (
              <div key={n.key} className="flex items-start gap-3 px-3 py-2.5">
                <span className="text-amber-500 mt-0.5 shrink-0 text-sm">⚠</span>
                <span className="text-xs text-muted-foreground flex-1">{n.message}</span>
                <a
                  href={`#${n.anchor}`}
                  className="text-xs text-amber-600 dark:text-amber-400 hover:underline shrink-0"
                >
                  Configure →
                </a>
              </div>
            ))}
          </div>
        )}

        {error && (
          <div className="mb-6 px-3 py-2 rounded-md bg-destructive/10 text-destructive text-xs">
            {error}
          </div>
        )}

        {/* Appearance */}
        <Section title="Appearance">
          <Field label="Theme">
            <div className="flex gap-2">
              {MODES.map(m => (
                <button
                  key={m.value}
                  onClick={() => setMode(m.value)}
                  className={`px-3 py-1.5 rounded-md text-sm border transition-colors ${
                    mode === m.value
                      ? "bg-accent text-accent-foreground border-transparent font-medium"
                      : "text-muted-foreground border-border hover:text-foreground"
                  }`}
                >
                  {m.label}
                </button>
              ))}
            </div>
          </Field>
          <Field label="Font size" hint="Cmd+  /  Cmd−  to adjust globally, Cmd+0 to reset">
            <div className="flex items-center gap-3">
              <button
                onClick={() => changeFont(fontSize - 1)}
                className="w-7 h-7 rounded-md border text-muted-foreground hover:text-foreground transition-colors text-sm"
              >−</button>
              <span className="text-sm font-mono w-10 text-center">{fontSize}px</span>
              <button
                onClick={() => changeFont(fontSize + 1)}
                className="w-7 h-7 rounded-md border text-muted-foreground hover:text-foreground transition-colors text-sm"
              >+</button>
              <button
                onClick={() => changeFont(FONT_DEFAULT)}
                className="text-xs text-muted-foreground hover:text-foreground transition-colors ml-1"
              >
                Reset
              </button>
            </div>
          </Field>
        </Section>

        {/* GitHub */}
        <Section id="section-github" title="GitHub">
          {configStatus.hasGitHubOAuth ? (
            <Field label="GitHub account" status="ok" statusLabel="connected">
              <div className="flex items-center gap-3">
                <span className="text-sm font-mono">{ghUser ? `@${ghUser}` : "authenticated"}</span>
                <button
                  onClick={disconnectGitHub}
                  disabled={ghLoading}
                  className="text-xs text-muted-foreground hover:text-destructive transition-colors disabled:opacity-50"
                >
                  {ghLoading ? "Disconnecting…" : "Disconnect"}
                </button>
              </div>
            </Field>
          ) : (
            <Field
              label="GitHub OAuth"
              hint="Connect via GitHub device flow to enable private repo access and PR creation"
              status="optional"
              statusLabel="not connected"
            >
              {wails ? (
                <button
                  onClick={() => setShowGitHubModal(true)}
                  className="px-3 py-1.5 rounded-md border text-sm hover:bg-accent transition-colors"
                >
                  Connect GitHub
                </button>
              ) : (
                <span className="text-xs text-muted-foreground">Available in the desktop app.</span>
              )}
            </Field>
          )}
          <Field
            id="field-github-token"
            label="GitHub token"
            hint="Personal access token (alternative to OAuth)"
            status={configStatus.hasGitHubToken ? "ok" : "optional"}
            statusLabel={configStatus.hasGitHubToken ? "configured" : "optional"}
          >
            <div className="flex gap-2">
              <SecretInput
                value={local.githubToken}
                onChange={e => set("githubToken", e.target.value)}
                placeholder="ghp_…"
                disabled={!wails}
              />
              {wails && configStatus.hasGitHubToken && (
                <button
                  onClick={testGitHub}
                  disabled={ghLoading}
                  className="px-3 py-1.5 rounded-md border text-xs hover:bg-accent transition-colors whitespace-nowrap disabled:opacity-50"
                >
                  {ghLoading ? "Testing…" : "Test"}
                </button>
              )}
            </div>
            {ghTestResult && (
              <p className={`text-xs mt-1 ${ghTestResult.ok ? "text-green-600 dark:text-green-400" : "text-destructive"}`}>
                {ghTestResult.ok ? `Connected as @${ghTestResult.login}` : "Connection failed — check token"}
              </p>
            )}
          </Field>
        </Section>

        {/* Model serving */}
        <Section title="Model Serving Endpoint">
          <Field label="Endpoint URL" hint="OpenAI-compatible base URL — e.g. https://openrouter.ai/api/v1 or https://api.openai.com/v1. Leave blank to use the built-in cluster LiteLLM.">
            <div className="flex gap-2">
              <TextInput
                value={local.litellmURL === "http://litellm:4000" ? "" : (local.litellmURL || "")}
                onChange={e => set("litellmURL", e.target.value || "http://litellm:4000")}
                placeholder="https://…/api/v1"
                disabled={!wails}
              />
              {wails && (
                <button
                  onClick={checkLiteLLM}
                  disabled={litellmChecking}
                  className="px-3 py-1.5 rounded-md border text-xs hover:bg-accent transition-colors whitespace-nowrap disabled:opacity-50"
                >
                  {litellmChecking ? "Checking…" : "Test"}
                </button>
              )}
            </div>
            {litellmResult && (
              <div className={`mt-2 text-xs ${litellmResult.ok ? "text-green-600 dark:text-green-400" : "text-destructive"}`}>
                {litellmResult.ok
                  ? `${litellmResult.models.length} model(s): ${litellmResult.models.join(", ")}`
                  : litellmResult.error || "Connection failed"}
              </div>
            )}
          </Field>
          <Field
            label="API key"
            hint="API key forwarded to the LLM provider (e.g. OpenRouter sk-or-…, OpenAI sk-…)"
            status={configStatus.hasLLMKey ? "ok" : "optional"}
            statusLabel={configStatus.hasLLMKey ? "configured" : "optional"}
          >
            <SecretInput
              value={local.llmApiKey ?? ""}
              onChange={e => set("llmApiKey", e.target.value)}
              placeholder="sk-…"
              disabled={!wails}
            />
          </Field>
        </Section>

        {/* Default Models */}
        <Section title="Default Models">
          <Field label="Manage phase" hint="Model used for the manage/planning agent">
            <TextInput
              value={local.defaultManageModel || ""}
              onChange={e => set("defaultManageModel", e.target.value)}
              placeholder="default"
              disabled={!wails}
            />
          </Field>
          <Field label="Implement phase" hint="Model used for the implement/coding agent">
            <TextInput
              value={local.defaultImplementModel || ""}
              onChange={e => set("defaultImplementModel", e.target.value)}
              placeholder="default"
              disabled={!wails}
            />
          </Field>
          <Field label="Copilot" hint="Model used for the copilot chat panel (⌘K). Leave blank to use the endpoint default.">
            <TextInput
              value={local.copilotModel ?? ""}
              onChange={e => set("copilotModel", e.target.value)}
              placeholder="default"
              disabled={!wails}
            />
          </Field>
        </Section>

        {/* Auto-update */}
        {wails && (
          <Section title="Updates">
            <Field label="Auto-update">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={local.autoUpdateEnabled || false}
                  onChange={e => set("autoUpdateEnabled", e.target.checked)}
                  className="rounded"
                />
                <span className="text-sm">Check for updates at launch</span>
              </label>
            </Field>
            <Field label="Channel" hint={local.updateChannel === "local" ? "Watches the installed app binary and reloads automatically when a new local build is installed." : undefined}>
              <div className="flex gap-2">
                {["stable", "nightly", "local"].map(ch => (
                  <button
                    key={ch}
                    onClick={() => set("updateChannel", ch)}
                    className={`px-3 py-1.5 rounded-md text-sm border transition-colors ${
                      (local.updateChannel || "stable") === ch
                        ? "bg-accent text-accent-foreground border-transparent font-medium"
                        : "text-muted-foreground border-border hover:text-foreground"
                    }`}
                  >
                    {ch.charAt(0).toUpperCase() + ch.slice(1)}
                  </button>
                ))}
              </div>
            </Field>
            <Field label="Version">
              <div className="flex items-center gap-3">
                <span className="text-sm text-muted-foreground font-mono">
                  {updateInfo
                    ? updateInfo.localBuild
                      ? "local build"
                      : updateInfo.currentVersion || "unknown"
                    : "—"}
                </span>
                <button
                  onClick={checkUpdate}
                  disabled={updateChecking}
                  className="text-xs text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
                >
                  {updateChecking ? "Checking…" : "Check now"}
                </button>
                {updateInfo && !updateInfo.localBuild && !updateInfo.upToDate && (
                  <a
                    href={updateInfo.releaseURL}
                    target="_blank"
                    rel="noreferrer"
                    className="text-xs text-accent-foreground hover:underline"
                  >
                    {updateInfo.latestVersion} available →
                  </a>
                )}
                {updateInfo && !updateInfo.localBuild && updateInfo.upToDate && (
                  <span className="text-xs text-green-600 dark:text-green-400">Up to date</span>
                )}
              </div>
            </Field>
          </Section>
        )}

        {/* Cluster connection */}
        <Section title="Cluster">
          <Field label="Namespace" hint="Kubernetes namespace where UNCWORKS is deployed">
            <TextInput
              value={local.namespace}
              onChange={e => set("namespace", e.target.value)}
              placeholder="uncworks"
              disabled={!wails}
            />
          </Field>
          <Field label="Kube context" hint="Leave blank to use the currently active context">
            <TextInput
              value={local.kubeContext}
              onChange={e => set("kubeContext", e.target.value)}
              placeholder="(active context)"
              disabled={!wails}
            />
          </Field>
        </Section>

        {/* Save — positioned after core sections */}
        <div className="mb-8 flex items-center gap-3">
          <button
            onClick={save}
            disabled={saving || !dirty}
            className="px-4 py-1.5 rounded-md bg-accent text-accent-foreground text-sm hover:opacity-90 disabled:opacity-40 transition-opacity"
          >
            {saved ? "Saved" : saving ? "Saving…" : "Save"}
          </button>
          {dirty && <span className="text-xs text-muted-foreground">Unsaved changes</span>}
          {!wails && (
            <span className="text-xs text-muted-foreground">
              Running in browser — settings saved to local storage only
            </span>
          )}
        </div>

        {/* Services — operational control, desktop only */}
        {wails && (
          <Section title="Services">
            <p className="text-xs text-muted-foreground mb-3">
              Services are automatically port-forwarded at startup. Restart or open as needed.
            </p>
            {services.length === 0 ? (
              <p className="text-xs text-muted-foreground">
                No services found — is the cluster running?
              </p>
            ) : (
              <div className="flex flex-col divide-y divide-border rounded-md border overflow-hidden mb-3">
                {services.filter(svc => {
                  // Hide litellm when using an external model serving endpoint
                  if (svc.name === "litellm") {
                    const url = local.litellmURL;
                    const isCluster = !url || url === "http://litellm:4000" || url.startsWith("http://localhost:");
                    return isCluster;
                  }
                  return true;
                }).map(svc => (
                  <ServiceRow
                    key={svc.name}
                    svc={svc}
                    restarting={restarting[svc.name]}
                    onRestart={() => restart(svc.name)}
                    onOpen={() => go().OpenService(svc.name)}
                  />
                ))}
              </div>
            )}
            <div className="flex items-center gap-4">
              <button
                onClick={loadOperational}
                className="text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                Refresh
              </button>
              <button
                onClick={() => go().OpenLogInConsole().catch(() => {})}
                className="text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                Open in Console.app
              </button>
            </div>
          </Section>
        )}

        {/* Advanced — collapsed by default */}
        <section className="mb-8">
          <button
            onClick={() => setAdvancedOpen(o => !o)}
            className="flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground uppercase w-full pb-1 border-b hover:text-foreground transition-colors mb-4"
          >
            <span>{advancedOpen ? "▾" : "▸"}</span>
            Advanced
          </button>

          {advancedOpen && (
            <>
              {/* Port range */}
              <p className="text-xs text-muted-foreground mb-3">
                Local port range for kubectl port-forward when accessing cluster services.
              </p>
              <div className="flex items-center gap-3 mb-6">
                <Field label="Start">
                  <NumberInput
                    value={local.portRangeStart}
                    onChange={e => set("portRangeStart", Number(e.target.value) || 0)}
                    min={1024}
                    max={65535}
                    disabled={!wails}
                  />
                </Field>
                <span className="text-muted-foreground mt-4">–</span>
                <Field label="End">
                  <NumberInput
                    value={local.portRangeEnd}
                    onChange={e => set("portRangeEnd", Number(e.target.value) || 0)}
                    min={1024}
                    max={65535}
                    disabled={!wails}
                  />
                </Field>
              </div>

              {/* Environment overrides */}
              <p className="text-xs text-muted-foreground mb-3">
                Override environment variables passed to subprocesses (kubectl, helm, uncworks).
                Blank inherits from the system shell.
              </p>
              {envVars.length === 0 && !wails && (
                <p className="text-xs text-muted-foreground mb-4">Available in the desktop app.</p>
              )}
              {envVars.map(ev => (
                <div key={ev.key} className="mb-3">
                  <div className="flex items-baseline gap-2 mb-0.5">
                    <label className="text-xs font-medium font-mono">{ev.key}</label>
                    {ev.system && (
                      <span className="text-xs text-muted-foreground truncate max-w-[200px]" title={ev.system}>
                        system: {ev.system}
                      </span>
                    )}
                  </div>
                  <input
                    type="text"
                    value={local.envOverrides?.[ev.key] ?? ""}
                    onChange={e => {
                      const val = e.target.value;
                      setLocal(s => ({
                        ...s,
                        envOverrides: { ...(s.envOverrides ?? {}), [ev.key]: val },
                      }));
                      setDirty(true);
                    }}
                    placeholder={ev.system || ev.desc}
                    disabled={!wails}
                    className="w-full px-2.5 py-1.5 rounded-md border bg-background text-sm font-mono focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
                  />
                </div>
              ))}

              {dirty && (
                <button
                  onClick={save}
                  disabled={saving}
                  className="mt-2 px-4 py-1.5 rounded-md bg-accent text-accent-foreground text-sm hover:opacity-90 disabled:opacity-40 transition-opacity"
                >
                  {saved ? "Saved" : saving ? "Saving…" : "Save"}
                </button>
              )}
            </>
          )}
        </section>
      </div>
      </div>
    </div>
  );
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function Section({ id, title, children }: { id?: string; title: string; children: React.ReactNode }) {
  return (
    <section id={id} className="mb-8 scroll-mt-4">
      <h2 className="text-xs font-semibold tracking-widest text-muted-foreground uppercase mb-4 border-b pb-1">
        {title}
      </h2>
      {children}
    </section>
  );
}

type FieldStatus = "ok" | "missing" | "optional";

function Field({
  id,
  label,
  hint,
  status,
  statusLabel,
  children,
}: {
  id?: string;
  label: string;
  hint?: string;
  status?: FieldStatus;
  statusLabel?: string;
  children: React.ReactNode;
}) {
  return (
    <div id={id} className="mb-4 scroll-mt-4">
      <div className="flex items-center gap-2 mb-1">
        <label className="text-xs font-medium">{label}</label>
        {status && statusLabel && (
          <span
            className={`text-xs px-1.5 py-0.5 rounded-full leading-none ${
              status === "ok"
                ? "bg-green-500/15 text-green-600 dark:text-green-400"
                : status === "missing"
                ? "bg-red-500/15 text-red-600 dark:text-red-400"
                : "bg-muted text-muted-foreground"
            }`}
          >
            {statusLabel}
          </span>
        )}
      </div>
      {children}
      {hint && <p className="text-xs text-muted-foreground mt-1">{hint}</p>}
    </div>
  );
}

function ServiceRow({
  svc,
  restarting,
  onRestart,
  onOpen,
}: {
  svc: ServiceInfo;
  restarting?: boolean;
  onRestart: () => void;
  onOpen: () => void;
}) {
  return (
    <div className="flex items-center gap-4 px-3 py-2.5 bg-background text-sm">
      <div className="flex items-center gap-2 w-36 shrink-0">
        <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${svc.ready ? "bg-green-500" : "bg-muted-foreground"}`} />
        <span className="truncate">{svc.displayName}</span>
      </div>
      <div className="flex-1 text-xs text-muted-foreground font-mono">
        {svc.forwarding
          ? `localhost:${svc.localPort} → :${svc.clusterPort}`
          : svc.clusterPort ? `:${svc.clusterPort}` : "—"}
      </div>
      <div className="flex items-center gap-2 shrink-0">
        {svc.clusterPort > 0 && svc.forwarding && (
          <Btn onClick={onOpen}>Open</Btn>
        )}
        <Btn onClick={onRestart} loading={restarting}>Restart</Btn>
      </div>
    </div>
  );
}

function TextInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      type="text"
      {...props}
      className="w-full px-2.5 py-1.5 rounded-md border bg-background text-sm font-mono focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
    />
  );
}

function NumberInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      type="number"
      {...props}
      className="w-28 px-2.5 py-1.5 rounded-md border bg-background text-sm font-mono focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
    />
  );
}

function SecretInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  const [show, setShow] = useState(false);
  return (
    <div className="relative flex items-center">
      <input
        {...props}
        type={show ? "text" : "password"}
        className="w-full px-2.5 py-1.5 pr-14 rounded-md border bg-background text-sm font-mono focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
      />
      {!props.disabled && (
        <button
          type="button"
          onClick={() => setShow(s => !s)}
          className="absolute right-2.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
          tabIndex={-1}
        >
          {show ? "hide" : "show"}
        </button>
      )}
    </div>
  );
}

function Btn({
  children,
  onClick,
  loading,
}: {
  children: React.ReactNode;
  onClick: () => void;
  loading?: boolean;
}) {
  return (
    <button
      onClick={onClick}
      disabled={loading}
      className="text-xs px-2 py-1 rounded border border-border text-muted-foreground hover:text-foreground hover:border-foreground transition-colors disabled:opacity-40"
    >
      {loading ? "…" : children}
    </button>
  );
}
