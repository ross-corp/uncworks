import { createSignal, For, Show } from "solid-js";
import { AOTClient } from "../../../packages/shared/src/grpc/client";
import type { AgentRun, AgentRunSpec, Backend } from "../../../packages/shared/src/types/agent-run";

interface CreateRunFormProps {
  client: AOTClient;
  onCreated: (run: AgentRun) => void;
}

interface RepoField {
  url: string;
  branch: string;
  path: string;
}

export default function CreateRunForm(props: CreateRunFormProps) {
  const [repos, setRepos] = createSignal<RepoField[]>([{ url: "", branch: "", path: "" }]);
  const [prompt, setPrompt] = createSignal("");
  const [backend, setBackend] = createSignal<Backend>("Pod");
  const [showAdvanced, setShowAdvanced] = createSignal(false);
  const [devboxConfig, setDevboxConfig] = createSignal("");
  const [ttlSeconds, setTtlSeconds] = createSignal("");
  const [image, setImage] = createSignal("");
  const [envVars, setEnvVars] = createSignal<{ key: string; value: string }[]>([]);
  const [submitting, setSubmitting] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [validationErrors, setValidationErrors] = createSignal<string[]>([]);

  function addRepo() {
    setRepos([...repos(), { url: "", branch: "", path: "" }]);
  }

  function updateRepo(index: number, field: keyof RepoField, value: string) {
    setRepos(repos().map((r, i) => (i === index ? { ...r, [field]: value } : r)));
  }

  function removeRepo(index: number) {
    if (repos().length > 1) {
      setRepos(repos().filter((_, i) => i !== index));
    }
  }

  function addEnvVar() {
    setEnvVars([...envVars(), { key: "", value: "" }]);
  }

  function updateEnvVar(index: number, field: "key" | "value", value: string) {
    setEnvVars(envVars().map((e, i) => (i === index ? { ...e, [field]: value } : e)));
  }

  function removeEnvVar(index: number) {
    setEnvVars(envVars().filter((_, i) => i !== index));
  }

  function validate(): boolean {
    const errors: string[] = [];
    if (!repos().some((r) => r.url.trim())) {
      errors.push("At least one repository URL is required");
    }
    if (!prompt().trim()) {
      errors.push("Prompt is required");
    }
    setValidationErrors(errors);
    return errors.length === 0;
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!validate()) return;

    setSubmitting(true);
    setError(null);

    const spec: AgentRunSpec = {
      backend: backend(),
      repos: repos()
        .filter((r) => r.url.trim())
        .map((r) => ({
          url: r.url.trim(),
          branch: r.branch.trim() || undefined,
          path: r.path.trim() || undefined,
        })),
      prompt: prompt().trim(),
      devboxConfig: devboxConfig().trim() || undefined,
      ttlSeconds: ttlSeconds() ? parseInt(ttlSeconds(), 10) : undefined,
      image: image().trim() || undefined,
      envVars: envVars().reduce(
        (acc, { key, value }) => {
          if (key.trim()) acc[key.trim()] = value;
          return acc;
        },
        {} as Record<string, string>
      ),
    };

    try {
      const run = await props.client.createAgentRun(spec);
      props.onCreated(run);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  }

  const inputStyle = { width: "100%", padding: "6px 8px", border: "1px solid #d1d5db", "border-radius": "4px", "box-sizing": "border-box" as const };
  const labelStyle = { display: "block", "font-size": "0.85em", color: "#374151", "margin-bottom": "4px", "font-weight": "bold" as const };

  return (
    <form onSubmit={handleSubmit} style={{ padding: "16px", border: "1px solid #e5e7eb", "border-radius": "8px", "margin-bottom": "16px", background: "#f9fafb" }}>
      <h3 style={{ "margin-top": 0 }}>New Agent Run</h3>

      <Show when={validationErrors().length > 0}>
        <div style={{ padding: "8px", background: "#fef2f2", "border-radius": "4px", "margin-bottom": "12px", color: "#991b1b" }}>
          <For each={validationErrors()}>{(err) => <div>{err}</div>}</For>
        </div>
      </Show>

      <Show when={error()}>
        <div style={{ padding: "8px", background: "#fef2f2", "border-radius": "4px", "margin-bottom": "12px", color: "#991b1b" }}>
          {error()}
        </div>
      </Show>

      {/* Repos */}
      <div style={{ "margin-bottom": "12px" }}>
        <label style={labelStyle}>Repositories</label>
        <For each={repos()}>
          {(repo, index) => (
            <div style={{ display: "flex", gap: "8px", "margin-bottom": "6px", "align-items": "center" }}>
              <input placeholder="URL *" value={repo.url} onInput={(e) => updateRepo(index(), "url", e.currentTarget.value)} style={{ ...inputStyle, flex: 3 }} />
              <input placeholder="Branch" value={repo.branch} onInput={(e) => updateRepo(index(), "branch", e.currentTarget.value)} style={{ ...inputStyle, flex: 1 }} />
              <input placeholder="Path" value={repo.path} onInput={(e) => updateRepo(index(), "path", e.currentTarget.value)} style={{ ...inputStyle, flex: 1 }} />
              <Show when={repos().length > 1}>
                <button type="button" onClick={() => removeRepo(index())} style={{ padding: "4px 8px", border: "1px solid #d1d5db", "border-radius": "4px", background: "transparent", cursor: "pointer" }}>×</button>
              </Show>
            </div>
          )}
        </For>
        <button type="button" onClick={addRepo} style={{ padding: "4px 12px", border: "1px solid #d1d5db", "border-radius": "4px", background: "transparent", cursor: "pointer", "font-size": "0.85em" }}>+ Add Repository</button>
      </div>

      {/* Prompt */}
      <div style={{ "margin-bottom": "12px" }}>
        <label style={labelStyle}>Prompt</label>
        <textarea value={prompt()} onInput={(e) => setPrompt(e.currentTarget.value)} rows={3} style={{ ...inputStyle, resize: "vertical" }} placeholder="What should the agent do?" />
      </div>

      {/* Backend */}
      <div style={{ "margin-bottom": "12px" }}>
        <label style={labelStyle}>Backend</label>
        <select value={backend()} onChange={(e) => setBackend(e.currentTarget.value as Backend)} style={inputStyle}>
          <option value="Pod">Pod</option>
          <option value="KubeVirt">KubeVirt</option>
          <option value="External">External</option>
        </select>
      </div>

      {/* Advanced */}
      <div style={{ "margin-bottom": "12px" }}>
        <button type="button" onClick={() => setShowAdvanced(!showAdvanced())} style={{ background: "transparent", border: "none", cursor: "pointer", color: "#6b7280", "font-size": "0.85em", padding: 0 }}>
          {showAdvanced() ? "▼" : "▶"} Advanced Options
        </button>
        <Show when={showAdvanced()}>
          <div style={{ "margin-top": "8px", "padding-left": "12px", "border-left": "2px solid #e5e7eb" }}>
            <div style={{ "margin-bottom": "8px" }}>
              <label style={labelStyle}>Devbox Config</label>
              <input value={devboxConfig()} onInput={(e) => setDevboxConfig(e.currentTarget.value)} style={inputStyle} placeholder="devbox.json content or URL" />
            </div>
            <div style={{ "margin-bottom": "8px" }}>
              <label style={labelStyle}>TTL (seconds)</label>
              <input type="number" value={ttlSeconds()} onInput={(e) => setTtlSeconds(e.currentTarget.value)} style={inputStyle} placeholder="3600" />
            </div>
            <div style={{ "margin-bottom": "8px" }}>
              <label style={labelStyle}>Image</label>
              <input value={image()} onInput={(e) => setImage(e.currentTarget.value)} style={inputStyle} placeholder="Custom agent image" />
            </div>
            <div style={{ "margin-bottom": "8px" }}>
              <label style={labelStyle}>Environment Variables</label>
              <For each={envVars()}>
                {(envVar, index) => (
                  <div style={{ display: "flex", gap: "8px", "margin-bottom": "4px" }}>
                    <input placeholder="KEY" value={envVar.key} onInput={(e) => updateEnvVar(index(), "key", e.currentTarget.value)} style={{ ...inputStyle, flex: 1 }} />
                    <input placeholder="VALUE" value={envVar.value} onInput={(e) => updateEnvVar(index(), "value", e.currentTarget.value)} style={{ ...inputStyle, flex: 2 }} />
                    <button type="button" onClick={() => removeEnvVar(index())} style={{ padding: "4px 8px", border: "1px solid #d1d5db", "border-radius": "4px", background: "transparent", cursor: "pointer" }}>×</button>
                  </div>
                )}
              </For>
              <button type="button" onClick={addEnvVar} style={{ padding: "4px 12px", border: "1px solid #d1d5db", "border-radius": "4px", background: "transparent", cursor: "pointer", "font-size": "0.85em" }}>+ Add Variable</button>
            </div>
          </div>
        </Show>
      </div>

      {/* Submit */}
      <button
        type="submit"
        disabled={submitting()}
        style={{ padding: "8px 20px", background: submitting() ? "#9ca3af" : "#10b981", color: "white", border: "none", "border-radius": "6px", cursor: submitting() ? "default" : "pointer", "font-weight": "bold" }}
      >
        {submitting() ? "Creating..." : "Create Run"}
      </button>
    </form>
  );
}
