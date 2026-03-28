## 1. Foundation — Config Bootstrap & Settings Struct

- [x] 1.1 Bootstrap `~/.config/uncworks/` on first launch in `cmd/uncworks-app/app.go` `startup()`
- [x] 1.2 Extend `AppSettings` struct: add `litellmURL`, `githubAuthed`, `updateChannel`, `autoUpdateEnabled`, `defaultManageModel`, `defaultImplementModel`; remove `llmKey`/`namespace` (autodetected)
- [x] 1.3 Add `GetKubeContexts() []string` and `AutodetectNamespace(ctx string) string` Wails bindings in `cmd/uncworks-app/app.go`
- [ ] 1.4 Migrate existing `config.json` on load: ignore `llmKey`, populate defaults for new fields

## 2. Go Dependencies

- [x] 2.1 Add `golang.org/x/oauth2` to `go.mod` (for GitHub device flow)
- [x] 2.2 Add `github.com/zalando/go-keyring` to `go.mod` (macOS Keychain)
- [x] 2.3 Run `go mod tidy`

## 3. GitHub Device Flow OAuth

- [x] 3.1 Implement `StartGitHubDeviceFlow() (userCode, verificationURL string, err error)` in `cmd/uncworks-app/github_auth.go`
- [x] 3.2 Implement `PollGitHubDeviceFlow(deviceCode string) (token string, done bool, err error)` — polls GitHub token endpoint
- [x] 3.3 Implement `SaveGitHubToken(token string)` → writes to Keychain service `uncworks`/account `github-token`
- [x] 3.4 Implement `GetGitHubUser() (login string, err error)` — reads token from Keychain, calls `GET api.github.com/user`
- [x] 3.5 Implement `DisconnectGitHub()` — removes token from Keychain, sets `githubAuthed: false`
- [x] 3.6 Expose above as Wails bindings

## 4. LiteLLM Config & Health

- [x] 4.1 Implement `CheckLiteLLM(url string) (models []string, err error)` in `cmd/uncworks-app/litellm.go` — calls `GET {url}/models`
- [x] 4.2 Add LiteLLM to services health check in `cmd/uncworks-app/health.go` using configured URL
- [x] 4.3 Expose `CheckLiteLLM` as Wails binding

## 5. Auto-Update

- [ ] 5.1 Embed build version via `-ldflags "-X main.Version=..."` in `wails build` call; add `Version` var to `cmd/uncworks-app/main.go`
- [ ] 5.2 Implement `CheckForUpdate() (UpdateInfo, error)` in `cmd/uncworks-app/update.go` — queries GitHub Releases API for latest stable or pre-release tag; returns new version string or empty if up-to-date
- [ ] 5.3 Local build detection: if `Version == ""` or `Version == "dev"`, return `UpdateInfo{LocalBuild: true}`
- [ ] 5.4 Cache update check result in-process for the session (don't re-query on every Settings visit)
- [ ] 5.5 Expose `CheckForUpdate` as Wails binding

## 6. Frontend — Setup Wizard

- [ ] 6.1 Create `web/src/components/SetupWizard.tsx` — full-screen modal overlay, 3-step progress indicator
- [ ] 6.2 Implement Step 1 (Cluster): kubecontext dropdown (from `GetKubeContexts`), namespace autodetect badge or manual fallback input
- [ ] 6.3 Implement Step 2 (GitHub): "Connect GitHub" button → device flow → show user code + URL → polling indicator → success/error state; "Skip" option
- [ ] 6.4 Implement Step 3 (LiteLLM): URL input (default `http://litellm:4000`), "Check connection" button calling `CheckLiteLLM`; model count badge or warning + "Skip for now"
- [ ] 6.5 Wire wizard completion to `save()` in `useSettings`; set `wizardComplete: true` in config
- [ ] 6.6 Show wizard automatically when `wizardComplete !== true` on app load (in `App.tsx` or `Layout.tsx`)

## 7. Frontend — Settings Page Restructure

- [ ] 7.1 Remove `llmKey` / "LLM API Key" field from `SettingsView.tsx`
- [ ] 7.2 Add GitHub auth section: show `@username` + Disconnect button (authenticated) or "Connect GitHub" → opens wizard at step 2 (unauthenticated)
- [ ] 7.3 Add LiteLLM section: URL input + "Test connection" button + model count badge
- [ ] 7.4 Add Auto-update section: enable toggle, channel selector (stable/nightly), "Check now" button, version status display
- [ ] 7.5 Add Default Models section: two model picker dropdowns (manage phase, implement phase) populated from LiteLLM `/models`
- [ ] 7.6 Add "Re-run setup wizard" CTA button at top of Settings page
- [ ] 7.7 Extend `AppSettings` type in `useSettings.tsx` to match new Go struct

## 8. Frontend — Run Submission Model Picker

- [ ] 8.1 Add model picker to run submission form (collapsible "Advanced" section)
- [ ] 8.2 Pre-fill manage/implement model pickers from `settings.defaultManageModel` / `settings.defaultImplementModel`
- [ ] 8.3 Pass selected models as fields in run submission payload

## 9. CSS — Overscroll Fix

- [ ] 9.1 Add `overscroll-behavior: none` to every `overflow-y-auto` scroll container in `web/src/` (SettingsView, run list, run detail, etc.)
- [ ] 9.2 Verify fix in Wails desktop app (rubber-band scroll eliminated)

## 10. Traffic Light UX Fixes

- [ ] 10.1 Increase top padding in Wails window chrome: bump `paddingTop` in `cmd/uncworks-app/wails.json` or the root layout div so content clears the traffic lights
- [ ] 10.2 Add `showTrafficLights` toggle to `AppSettings` struct
- [ ] 10.3 Wire toggle to Wails `WindowSetSystemDefaultTheme` or `WindowHideSystemDefaultMenuBar`; conditionally use `TitleBarHiddenInset` vs `TitleBarHidden` in `mac.TitleBar`

## 11. Sidebar Navigation Grouping

- [ ] 11.1 Restructure `NAV_ITEMS` array in `GlobalNav.tsx` into `NAV_GROUPS: { label: string; items: NavItem[] }[]` with three groups: Activity (Runs), Library (Projects, Templates), Automation (Chains, Schedules)
- [ ] 11.2 Render group labels (muted, 10px, tracking-widest, uppercase) when sidebar is expanded; hide labels when collapsed
- [ ] 11.3 Add 8px gap between groups in both expanded and collapsed modes via `mt-2` margin on non-first groups
- [ ] 11.4 Update `configIncomplete` condition in `GlobalNav.tsx` to only check `!configStatus.hasGitHubToken` (remove `!configStatus.hasLLMKey` since llmKey is being removed)
