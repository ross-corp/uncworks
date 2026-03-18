## Context

Agent runs are identified by K8s-safe names like `ar-a3gfp3`. Users see these names in the run list, detail header, and command palette. With more than a handful of runs, users cannot distinguish runs without reading each prompt. A human-readable name derived from the prompt would make the UI navigable at a glance.

## Goals / Non-Goals

**Goals:**
- Generate a short, descriptive, kebab-case name from the run's prompt at creation time
- Store the name as `display_name` on the AgentRunSpec (proto and CRD)
- Show `display_name` throughout the web UI with graceful fallback to the K8s name
- Handle LLM unavailability without blocking run creation

**Non-Goals:**
- Allowing users to manually set or edit the display name (future enhancement)
- Deduplicating display names across runs (they are not used as identifiers)
- Replacing the K8s resource name — `ar-{random}` remains the canonical ID
- Using a larger or external LLM model for name generation

## Decisions

### 1. Call Ollama via the existing LiteLLM proxy

The API server already knows the LiteLLM proxy URL (used for agent LLM access). The name generation call uses the same endpoint with model `qwen2.5:0.5b` (already deployed in-cluster via Ollama). This avoids adding a new dependency or service URL.

**System prompt:** `"Generate a short kebab-case name (3-5 words) for this task. Output ONLY the name, nothing else."`

The user prompt is the run's prompt text, truncated to 200 characters if longer (the model only needs the gist).

**Alternative considered:** Call Ollama directly, bypassing LiteLLM. Rejected — the API server already has the LiteLLM URL configured, and LiteLLM handles model routing uniformly.

### 2. Add `display_name` to AgentRunSpec (not status)

The display name is a spec-level field because it describes the run's intent, set at creation time, and never changes. It is not derived from runtime state.

The proto message gets `string display_name = N` added to `AgentRunSpec`. The CRD Go type gets `DisplayName string` in the spec struct with `json:"displayName,omitempty"`.

### 3. K8s resource name stays as `ar-{random}`

K8s names have strict constraints (63 chars, DNS-safe, unique per namespace). The display name is for humans only — it has no uniqueness constraint and no role in K8s resource identification. Keeping K8s names as `ar-{random}` avoids name collision handling and DNS issues.

### 4. Regex validation with fallback

Generated names are validated against `^[a-z0-9][a-z0-9-]{2,48}[a-z0-9]$`. This ensures:
- Starts and ends with alphanumeric
- Only lowercase letters, digits, and hyphens
- Length between 4 and 50 characters

If the LLM returns an invalid name (fails regex, empty, or contains unexpected content), the API server falls back to leaving `display_name` empty. The UI then shows the K8s name.

If the LLM call fails entirely (timeout, connection refused, 5xx), the API server logs a warning and proceeds with an empty `display_name`. Run creation is never blocked by name generation.

**Timeout:** The LLM call has a 3-second timeout. The qwen2.5:0.5b model generates a few tokens nearly instantly; 3 seconds is generous.

### 5. UI shows display_name with K8s name fallback

All UI components that show the run name use a helper: `displayName(run) => run.spec.displayName || run.metadata.name`. This is applied in:
- **RunList**: Primary column shows display name, K8s name shown as secondary/muted text
- **RunDetail header**: Title shows display name, subtitle shows K8s name
- **CommandPalette**: Searches both display name and K8s name, displays display name as the primary label

## Risks / Trade-offs

- **[Risk] LLM generates inappropriate or unhelpful names** — The qwen2.5:0.5b model is small and may produce poor names for ambiguous prompts. Acceptable — the K8s name is always visible as secondary text, and users can identify runs by either name. A future enhancement could allow manual renaming.
- **[Risk] LLM adds latency to run creation** — Mitigated by the 3-second timeout and the fact that qwen2.5:0.5b is fast (~100ms for a few tokens). If it exceeds 3s, we fall back silently.
- **[Risk] Display name not unique** — Two runs with similar prompts may get the same display name. This is fine — display names are not identifiers, and the K8s name provides uniqueness.
- **[Trade-off] Proto field is additive** — Old clients ignore `display_name`. No breaking change.
