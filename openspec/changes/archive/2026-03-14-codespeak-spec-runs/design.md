## Context

Agent runs are currently triggered by a free-text prompt. The agent receives the prompt and does whatever it interprets from it. CodeSpeak introduces a more structured paradigm — `.cs.md` markdown specs that describe *what* code should do, and `codespeak build` compiles those specs into actual code.

The web UI currently uses a `<textarea>` for prompt input. There's no rich editor, no file management, and no way to trigger runs from external events like a git push.

CodeSpeak specs use a `codespeak.json` config that references spec files and whitelisted files. The `codespeak build` CLI handles spec parsing, code generation, and testing. The build process requires an Anthropic API key (BYOK model).

## Goals / Non-Goals

**Goals:**
- Embed Monaco editor in the web UI for authoring `.cs.md` specs with markdown highlighting
- Support a "spec run" mode where the agent receives a CodeSpeak spec and runs `codespeak build`
- Store spec content on the AgentRun so specs are retrievable and re-runnable
- Push specs to GitHub repos (commit + push the `.cs.md` file)
- Pull specs from GitHub repos (load spec content from a file in a repo)
- Accept webhook triggers that spawn spec runs when `.cs.md` files change in a repo
- Make spec-driven and prompt-driven runs coexist cleanly

**Non-Goals:**
- Building a full CodeSpeak IDE (project management, multi-file specs, `codespeak.json` editing) — we're integrating single-spec authoring and runs
- Running `codespeak build` on the control plane — the agent pod does the build
- Managing CodeSpeak API keys — the agent's LLM key or user-provided env vars handle this
- Real-time collaborative editing of specs — single-user editor for now

## Decisions

### 1. Monaco editor integration via @monaco-editor/react

**Decision**: Use the `@monaco-editor/react` wrapper package to embed Monaco editor. Configure it with markdown language mode, the VS Code dark theme (matching our UI), and a fixed set of editor options (minimap off, word wrap on, line numbers on).

**Rationale**: `@monaco-editor/react` handles the Monaco lifecycle (loading, disposal, resizing) and is the standard React integration. Markdown mode gives us syntax highlighting for CodeSpeak's `.cs.md` format out of the box since CodeSpeak specs are valid markdown. No need for a custom language grammar.

**Alternative considered**: CodeMirror — lighter weight but less feature-rich. Monaco's familiarity (it's VS Code) is a UX advantage for developers.

### 2. Spec content stored on AgentRunSpec

**Decision**: Add two new fields to the proto `AgentRunSpec`:

```protobuf
string spec_content = 13;   // The .cs.md spec body (markdown)
string spec_source = 14;    // Where the spec came from: "editor", "github:<owner/repo/path>", etc.
```

When `spec_content` is non-empty, it's a spec-driven run. The prompt field still exists and is auto-generated from the spec (e.g., "Run `codespeak build` with the attached spec"). The spec content is written to the workspace as a `.cs.md` file by the hydrator before the agent starts.

**Rationale**: Storing spec content on the run makes it self-contained — you can view, re-edit, and re-run any past spec run. The `spec_source` field provides provenance tracking. Using existing proto fields avoids a separate spec storage system.

**Alternative considered**: Storing specs in a separate CRD/database — rejected as premature. The spec is an input to the run, not an independent entity (yet).

### 3. Agent run form: dual-mode (prompt vs. spec)

**Decision**: The agent run form gets a tab selector at the top: **Prompt** | **Spec**. The Prompt tab shows the existing textarea. The Spec tab shows the Monaco editor. Only one is active per run. The form submits either `prompt` (prompt mode) or `spec_content` + auto-generated prompt (spec mode).

```
┌─────────────────────────────────────────┐
│  New Agent Run                      ×   │
├─────────────────────────────────────────┤
│  Name: [fix-eml-converter__________]   │
│  Workspace: [payments-platform ▾]       │
│                                         │
│  ┌──────────┐ ┌──────────┐             │
│  │  Prompt  │ │   Spec   │ ◄── active  │
│  └──────────┘ └──────────┘             │
│  ┌─────────────────────────────────┐   │
│  │ # EmlConverter                  │   │
│  │                                 │   │
│  │ Converts RFC 5322 email files   │   │
│  │ (.eml) to Markdown.             │   │
│  │                                 │   │
│  │ ## Accepts                      │   │
│  │ - .eml files                    │   │
│  │                                 │   │
│  │ ## Output Structure             │   │
│  │ ...                             │   │
│  └─────────────────────────────────┘   │
│                                         │
│  Backend [Pod ▾]  Model [Premium ▾]     │
│                                         │
│              [Cancel] [Create Run]      │
└─────────────────────────────────────────┘
```

**Rationale**: Tabs cleanly separate the two modes without cluttering the form. The Monaco editor replaces the textarea entirely in spec mode — no split-pane complexity.

### 4. Spec file placement in workspace

**Decision**: When a spec-driven run starts, the hydrator writes the spec content to `/workspace/spec/main.cs.md` and generates a minimal `/workspace/codespeak.json`:

```json
{
  "specs": ["spec/main.cs.md"]
}
```

The agent's prompt tells it to run `codespeak build` in `/workspace`.

**Rationale**: This mirrors CodeSpeak's standard project layout. The agent doesn't need to know about the UI or API — it just sees a CodeSpeak project and builds it. For mixed-mode runs (spec + existing repos), the spec and `codespeak.json` are placed at the workspace root alongside the cloned repos.

### 5. GitHub sync: push and pull

**Decision**: Two mechanisms:

**Push (UI → GitHub)**: When a user saves a spec in the UI, they can optionally push it to a GitHub repo. This creates a commit with the `.cs.md` file at a specified path. Implemented as a new API endpoint that uses a GitHub App or PAT to commit.

**Pull (GitHub → UI)**: The spec editor can load a spec from a GitHub repo by path. The API fetches the file content via GitHub API and populates the editor.

Both use the GitHub API (via `gh` CLI or direct API calls from the server), not git clone.

**Rationale**: Pushing/pulling individual files via the API is lighter than cloning repos. Specs are small text files — no need for full git operations. The GitHub App approach lets the platform act on behalf of the user without storing personal tokens.

### 6. Webhook trigger for spec runs

**Decision**: Add a new HTTP endpoint `POST /api/v1/webhooks/github` that receives GitHub push webhooks. When a push contains changes to `.cs.md` files, the webhook handler:

1. Reads the updated spec content from GitHub
2. Creates an AgentRun with `spec_content` set and `spec_source` pointing to the repo/file
3. Uses the repo as the workspace (adds it to `repos[]`)

The webhook requires a shared secret for verification (standard GitHub webhook signature validation).

**Rationale**: This closes the loop — specs authored in a repo (via any editor/IDE) automatically trigger agent runs. No UI interaction needed. Developers can use their preferred editor, commit, and the platform picks it up.

**Alternative considered**: Polling repos for changes — rejected because webhooks are immediate and standard. GitHub Actions trigger — possible but adds a dependency on the user's CI config.

### 7. Spec display in detail panel and table

**Decision**: Spec runs show a "View Spec" button in the detail panel that opens the Monaco editor in read-only mode. The table shows a small indicator (icon or badge) distinguishing spec runs from prompt runs. The spec source (e.g., "github:org/repo/path.cs.md") is shown in the detail panel metadata.

**Rationale**: Specs are the primary artifact of spec runs — they should be viewable with the same rich editor used to create them.

## Risks / Trade-offs

**Monaco bundle size** — Monaco adds ~2MB to the JS bundle (gzipped). → Mitigation: Lazy-load the editor component. Only import Monaco when the user opens the spec tab or views a spec. Code-split via dynamic `import()`.

**CodeSpeak availability in agent pods** — The agent needs `codespeak` CLI installed. → Mitigation: Add `codespeak` to the agent container image, or have the agent install it via devbox (`devbox add codespeak`). If using devbox, the workspace's `devbox.json` should include it.

**GitHub App permissions** — Pushing commits requires write access. → Mitigation: Use a GitHub App with scoped permissions (contents: write on specific repos). Users authorize the app per-repo. Start with PAT-based auth for simplicity, migrate to GitHub App later.

**Spec + prompt confusion** — Users might not understand when to use specs vs. prompts. → Mitigation: Specs are opt-in (tab selection). The default remains prompt mode. Add a brief description on the spec tab explaining what CodeSpeak specs are.

**Webhook security** — Public webhook endpoints are attack surfaces. → Mitigation: Standard GitHub webhook signature validation (HMAC-SHA256). Rate limiting. Only process events from configured repos.
