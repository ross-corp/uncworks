## Documentation Structure

```
docs/
├── README.md                  ← "What is UNCWORKS" (synced to wiki Home)
├── getting-started.md         ← Quick start with k0s local dev
├── architecture/
│   ├── overview.md            ← High-level system diagram
│   ├── control-plane.md       ← API server, controller, Temporal worker
│   ├── agent-pods.md          ← Sidecar, pi, determinism extension
│   ├── pipeline.md            ← Plan → Execute → Verify flow
│   └── workspace.md           ← Hydration, workspace layout, OpenSpec
├── guides/
│   ├── creating-runs.md       ← Creating and managing runs via UI
│   ├── spec-driven.md         ← Spec-driven pipeline deep dive
│   └── models.md              ← LiteLLM config, model selection
├── reference/
│   ├── api.md                 ← ConnectRPC API reference
│   ├── crd.md                 ← AgentRun CRD fields
│   ├── extension.md           ← Determinism extension reference
│   └── helm-values.md         ← Helm chart values
└── contributing/
    ├── development.md         ← Local dev setup (k0s, Taskfile)
    └── testing.md             ← Test infrastructure
```

## Wiki Sync

```
┌──────────────┐  push to main   ┌──────────────────┐
│  docs/ in    │────────────────▶│  GitHub Actions   │
│  main repo   │                 │  wiki-sync.yml    │
└──────────────┘                 └────────┬─────────┘
                                          │
                                          │ git push
                                          ▼
                                 ┌──────────────────┐
                                 │  Wiki repo        │
                                 │  ross-corp/       │
                                 │  uncworks.wiki    │
                                 └──────────────────┘
```

The workflow:
1. Triggers on push to `main` when `docs/**` changes
2. Clones the wiki repo (`ross-corp/uncworks.wiki.git`)
3. Copies `docs/**/*.md` files, flattening paths for wiki page names:
   - `docs/architecture/overview.md` → `Architecture-Overview.md`
   - `docs/guides/spec-driven.md` → `Guides-Spec-Driven.md`
   - `docs/README.md` → `Home.md` (wiki landing page)
4. Generates `_Sidebar.md` from the directory structure
5. Commits and pushes to wiki repo

## Doc Staleness Detection

A CI check that runs on PRs modifying Go or TypeScript source:

1. Extracts all backtick-quoted identifiers from docs (function names, type names, file paths)
2. Greps for them in the source code
3. Reports any that no longer exist as warnings
4. Fails the check if >5 stale references found (configurable threshold)

Implementation: a shell script (`scripts/check-doc-staleness.sh`) invoked by a GitHub Action.

## Architecture Diagrams

All diagrams use Mermaid (rendered natively by GitHub). Key diagrams:

### System Overview
```
┌─────────────────────────────────────────────────────────┐
│                    UNCWORKS                               │
│                                                          │
│  ┌──────────┐   ┌───────────┐   ┌──────────────────┐   │
│  │ Web UI   │──▶│ API Server│──▶│ Temporal Workflow │   │
│  │ React    │   │ ConnectRPC│   │ (spec-driven)    │   │
│  └──────────┘   └───────────┘   └────────┬─────────┘   │
│                                           │              │
│                       ┌───────────────────▼────────┐    │
│                       │       Agent Pod             │    │
│                       │  ┌─────────┐  ┌─────────┐  │    │
│                       │  │ pi      │  │ sidecar │  │    │
│                       │  │ (agent) │  │ (RPC gw)│  │    │
│                       │  └─────────┘  └─────────┘  │    │
│                       └────────────────────────────┘    │
│                                                          │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐            │
│  │ LiteLLM  │   │ Ollama   │   │ Temporal │            │
│  │ (proxy)  │   │ (local)  │   │ (server) │            │
│  └──────────┘   └──────────┘   └──────────┘            │
└─────────────────────────────────────────────────────────┘
```

### Pipeline Flow
```
User prompt
    │
    ▼
┌─────────────────┐
│  agent-manage    │  PI_ROLE=manage
│  (plan stage)    │
│  • reads repo    │
│  • openspec CLI  │
│  • writes specs  │
│  • validates     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ agent-implement  │  PI_ROLE=implement
│ (execute stage)  │
│  • reads specs   │
│  • writes code   │
│  • runs tests    │
│  • marks tasks   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  agent-manage    │  PI_ROLE=manage
│  (verify stage)  │
│  • openspec      │
│    validate      │
│  • task check    │
│  • LLM judge     │
│  • archive       │
└────────┬────────┘
         │
    pass? ──no──▶ retry execute
         │
        yes
         │
         ▼
    ✓ SUCCEEDED
```

## Branding Changes

| Location | Current | New |
|----------|---------|-----|
| README.md title | "AOT -- Agent Orchestration Tool" | "UNCWORKS" |
| README.md subtitle | "A Cloud Native OS for AI Engineers" | "An agentic development environment" |
| HTML `<title>` | "AOT — UNCWERKS" | "UNCWORKS" |
| Helm chart NOTES.txt | "AOT has been installed" | "UNCWORKS has been installed" |
| docs/ headers | "AOT Architecture", "AOT API" | "UNCWORKS Architecture", etc. |
| ROADMAP.md | "AOT Roadmap" | "UNCWORKS Roadmap" |

Internal code (`aot-controller`, `aot.uncworks.io`, proto packages) stays unchanged.
