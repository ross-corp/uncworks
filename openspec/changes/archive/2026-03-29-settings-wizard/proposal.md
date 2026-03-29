## Why

UNCWORKS has no first-run setup flow: the config directory (`~/.config/uncworks/`) is never created, required credentials are never collected, and users drop into a blank UI with no guidance. The Settings page is also overloaded — it mixes connection management, advanced tuning, and credential entry in a single flat form with no hierarchy.

## What Changes

- **Setup wizard** — multi-step first-run flow (re-runnable from Settings) covering: cluster autodetection, GitHub OAuth (device flow), and LiteLLM endpoint configuration with provider health check
- **GitHub Device Flow OAuth** — replace raw PAT input with in-app device flow using the `gh` CLI client_id; store token in macOS Keychain; show auth status inline
- **LiteLLM configurability** — user-configurable LiteLLM URL (default: `http://litellm:4000`), configurable to any OpenAI-compatible endpoint; provider readiness shown as a wizard gate
- **Namespace autodetection** — remove namespace as a required manual field; autodetect from active cluster context (walk `kubectl get ns` for `uncworks`)
- **LiteLLM in services list** — add LiteLLM to the health/services panel (autodiscovered via cluster labels or configured URL)
- **Auto-update** — opt-in update mechanism with nightly/stable channel selection; version detection; local build UX (no update available, show build SHA)
- **Default model settings** — configurable default model for manage-phase and implement-phase agents separately
- **Model selection per run** — model picker in run submission UI (uses defaults, overridable per run)
- **Rubber-band scroll fix** — `overscroll-behavior: none` on all scroll containers in the Wails app
- **Config directory bootstrap** — create `~/.config/uncworks/` on first launch; trigger wizard if config is absent
- **Sidebar navigation grouping** — restructure flat `NAV_ITEMS` into labelled groups (Activity, Library, Automation); labels visible when expanded, hidden when collapsed; `configIncomplete` check scoped to GitHub token only

## Capabilities

### New Capabilities

- `setup-wizard`: Multi-step first-run setup wizard (cluster → GitHub → LiteLLM); re-runnable from Settings; autodetection for cluster, namespace, and LiteLLM
- `github-oauth`: Device-flow GitHub OAuth using `gh` CLI client_id; Keychain storage; in-app auth status
- `litellm-config`: User-configurable LiteLLM URL; provider setup check as wizard gate; in-cluster default with external override
- `auto-update`: Opt-in update check; nightly vs stable channel; local build detection
- `default-model-settings`: Configurable default model per agent phase (manage, implement); overridable per run in submission UI

### Modified Capabilities

- `model-selection-ui`: Model picker now appears in run submission UI (not just settings); draws from defaults but allows per-run override
- `cluster-management`: Namespace is now autodetected rather than a required manual field

## Impact

- **`cmd/uncworks-app/`**: New wizard flow, GitHub device flow OAuth, Keychain integration, auto-update checker
- **`cmd/uncworks-app/settings.go`**: `AppSettings` struct expanded (LiteLLM URL, GitHub OAuth token, update channel, default models); config directory bootstrap
- **`web/src/views/SettingsView.tsx`**: Restructured around wizard-first UX; surfaces wizard re-run CTA
- **`web/src/views/NewRunView.tsx`** (or run submission form): Model picker added
- **`web/src/hooks/useSettings.tsx`**: Extended `AppSettings` and `ConfigStatus` for new fields
- **`web/src/index.css`**: `overscroll-behavior: none` on all scroll containers
- **Go deps**: `golang.org/x/oauth2` (device flow), `github.com/zalando/go-keyring` (Keychain)
