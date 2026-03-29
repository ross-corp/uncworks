## Context

The Wails desktop app has no first-run setup flow. `~/.config/uncworks/` is never created until the user saves Settings manually — but they can't meaningfully save settings because they don't know what to fill in. The current settings form is a flat JSON-key dump: `llmKey` (architecturally wrong — the app doesn't call LLMs directly), `namespace` (should autodetect), raw PAT field (should be OAuth), and no guidance for LiteLLM.

Current state:
- `cmd/uncworks-app/settings.go`: XDG-aware path, `AppSettings` struct, `GetSettings`/`SaveSettings` Wails bindings
- `web/src/hooks/useSettings.tsx`: Context, `ConfigStatus`, Wails/localStorage dual path
- `web/src/views/SettingsView.tsx`: Flat form, no wizard UX
- No GitHub OAuth, no Keychain, no auto-update, no model defaults

## Goals / Non-Goals

**Goals:**
- Bootstrap `~/.config/uncworks/` on first launch; trigger wizard when config is absent
- Wizard: 3 focused steps — Cluster (autodetect kubecontext + namespace) → GitHub (device flow OAuth) → LiteLLM (URL + provider check)
- Re-runnable wizard from Settings page
- Configurable LiteLLM URL (default `http://litellm:4000`); provider health check as wizard gate
- GitHub device flow via `gh` CLI `client_id`; token in macOS Keychain via `go-keyring`
- Auto-update: opt-in, nightly/stable channel, local build detection (no-op for non-tagged builds)
- Default model per phase (manage, implement); model picker in run submission UI
- `overscroll-behavior: none` on all scroll containers

**Non-Goals:**
- Code signing or notarization (out of scope for now)
- Windows/Linux auto-update (macOS only first)
- Multi-account GitHub (single token)
- LiteLLM provider provisioning (only checks if providers exist, doesn't create them)

## Decisions

### D1: Wizard as a modal overlay, not a route

The wizard runs as a full-screen modal overlay on first launch (and when re-triggered from Settings). This avoids route management complexity, allows the wizard to appear before the app shell is fully rendered, and lets us dismiss it and return to current context.

**Alternative**: Separate `/setup` route. Rejected — adds router state management, harder to re-trigger from settings button.

### D2: GitHub Device Flow with `gh` CLI client_id

Use `golang.org/x/oauth2/github` device flow with the `gh` CLI's public client_id (`178c6fc778ccc68e1d6a`). The user sees a device code + URL, approves in browser, app polls for token.

**Alternative**: Register our own OAuth App. Rejected — requires app distribution credentials, more setup friction, no benefit for this use case.

Token stored via `github.com/zalando/go-keyring` → macOS Keychain service `uncworks`, account `github-token`. Falls back to `~/.config/uncworks/github-token` on Linux (not in scope yet).

### D3: LiteLLM URL in settings, provider check as wizard gate

Wizard step 3 shows a URL field (default `http://litellm:4000`). After URL entry, the wizard calls `GET /models` on LiteLLM to list available providers. If zero providers, shows a warning with a link to LiteLLM docs — user can skip or configure before proceeding.

**Alternative**: Auto-discover LiteLLM from cluster labels. Deferred — adds k8s dependency to wizard flow before cluster step is complete.

### D4: Namespace autodetection via kubeconfig context

On wizard step 1, run `kubectl get ns -o json` (using the selected kubecontext), filter for namespaces matching `uncworks` or having label `app.kubernetes.io/part-of=uncworks`. If exactly one match, auto-select. If multiple or zero, show a dropdown.

### D5: Auto-update via GitHub Releases API

For stable channel: check `api.github.com/repos/uncworks/uncworks/releases/latest`. For nightly: check `releases` and filter for pre-release tags matching `v*-pre.*`. Compare against embedded build version (set at `wails build` time via `-ldflags`). Local builds (no tag) show "local build — updates not available".

**Alternative**: Sparkle framework (macOS native). Deferred — requires code signing infrastructure.

### D6: Model defaults as two fields, not a global default

`defaultManageModel` and `defaultImplementModel` are separate fields. The run submission UI shows both with pre-filled values from defaults, each independently overridable per run.

### D7: `overscroll-behavior: none` via CSS class

Add `.no-overscroll { overscroll-behavior: none; }` utility and apply to every `overflow-y-auto` container. The `html[data-wails]` global rule handles the root but not inner containers.

## Risks / Trade-offs

- **`gh` client_id misuse** → GitHub may rate-limit or revoke if heavily used outside `gh`. Mitigation: we use it the same way as `gh` CLI (device flow only, no scope creep). If revoked, register own client_id.
- **Keychain prompts on macOS** → `go-keyring` triggers a Keychain access dialog on first write. Mitigation: acceptable UX; explain to user in wizard step.
- **LiteLLM provider check false-negatives** → If LiteLLM is temporarily down during wizard, step 3 will show a warning. Mitigation: "Skip for now" option; re-runnable wizard.
- **kubeconfig context not available** → If no kubeconfig or no context active, cluster step should gracefully degrade to manual entry rather than crashing.
- **Auto-update GitHub API rate limits** → Unauthenticated requests limited to 60/hr. Mitigation: check at most once per launch, cache result for session.

## Migration Plan

1. Extend `AppSettings` struct — new fields have zero-value defaults, no migration needed for existing configs
2. On app launch: if `~/.config/uncworks/config.json` absent → show wizard modal
3. Wizard completion writes config; subsequent launches skip wizard
4. GitHub token migration: if existing `githubToken` field is populated, skip GitHub step in wizard (already configured)

## Open Questions

- Should the wizard have a "check all" summary step, or is 3 steps sufficient?
- For auto-update download+install: open browser to GitHub Releases page, or attempt in-app download? (Currently leaning: open browser — avoids Gatekeeper complexity without signing)
