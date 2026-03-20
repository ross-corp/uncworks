## Architecture

### Agent Role Model

```
┌────────────────────────────────────────────────────────────┐
│                        User (browser)                       │
│                             │                               │
│                        ┌────▼────┐                          │
│                        │  Web UI │                          │
│                        └────┬────┘                          │
│                             │                               │
│                   ┌─────────▼──────────┐                    │
│                   │   agent-manage     │ ← persistent       │
│                   │   (orchestrator)   │   across stages    │
│                   │                    │                     │
│                   │  Tools:            │                     │
│                   │   ✓ openspec CLI   │                     │
│                   │   ✓ ask_user       │                     │
│                   │   ✓ spawn_agent    │                     │
│                   │   ✓ read (repo)    │                     │
│                   │   ✗ write (repo)   │                     │
│                   │   ✗ bash           │                     │
│                   └─────────┬──────────┘                    │
│                             │ spawns                        │
│              ┌──────────────▼──────────────┐                │
│              │      agent-implement        │                │
│              │      (worker, ephemeral)    │                │
│              │                             │                │
│              │  Tools:                     │                │
│              │   ✓ read/write/edit (repo)  │                │
│              │   ✓ bash                    │                │
│              │   ✓ spawn_agent (sub-impl)  │                │
│              │   ✗ openspec CLI            │                │
│              │   ✗ ask_user (escalate)     │                │
│              └─────────────────────────────┘                │
└────────────────────────────────────────────────────────────┘
```

### Workspace Layout

```
/workspace/                          ← workspace root
├── .openspec/                       ← OpenSpec artifacts (workspace-level)
│   ├── config.yaml
│   └── changes/
│       └── ar-xyz123/
│           ├── proposal.md
│           ├── design.md
│           ├── specs/<cap>/spec.md
│           └── tasks.md
├── .aot/                            ← AOT runtime artifacts
│   ├── logs/
│   ├── input/
│   └── traces/
├── .bare/                           ← bare git repos (hidden)
│   └── neph.nvim/
├── neph.nvim/                       ← git worktree (repo at root level)
│   ├── lua/
│   ├── tests/
│   └── ...
└── another-repo/                    ← second repo (multi-repo runs)
    └── ...
```

Key changes from current layout:
- Repos at `/workspace/<repo-name>/` not `/workspace/src/<repo-name>/`
- OpenSpec at `/workspace/.openspec/` not inside repo's `openspec/`
- Single-repo runs: repo IS the workspace root (worktree directly into `/workspace/`)
  - This requires `git worktree add /workspace/<repo-name>` not `/workspace/` (can't worktree into existing dir)
  - So even single repos get `/workspace/<repo-name>/`

### Agent Lifecycle (Spec-Driven Pipeline)

```
Phase 1: PLAN
┌─────────────────────────────────────┐
│  agent-manage starts                │
│  ├─ reads repo to understand scope  │
│  ├─ runs openspec CLI commands      │
│  │   (init, instructions, validate) │
│  ├─ writes proposal, specs, tasks   │
│  ├─ may ask_user for clarification  │
│  └─ validates specs pass            │
└─────────────────────────────────────┘
                │
Phase 2: EXECUTE
┌─────────────────────────────────────┐
│  agent-manage spawns agent-implement│
│  ├─ agent-implement reads specs     │
│  ├─ implements code changes         │
│  ├─ marks tasks as [x]             │
│  ├─ may spawn sub-agents for        │
│  │   parallel work                  │
│  └─ exits when done                 │
│                                     │
│  agent-manage monitors progress     │
│  ├─ can steer/interrupt implement   │
│  └─ can spawn additional workers    │
└─────────────────────────────────────┘
                │
Phase 3: VERIFY
┌─────────────────────────────────────┐
│  agent-manage evaluates results     │
│  ├─ runs openspec validate          │
│  ├─ checks task completion          │
│  ├─ runs test commands from specs   │
│  ├─ if FAIL: spawns new implement   │
│  │   agent with failure context     │
│  └─ if PASS: archives change        │
└─────────────────────────────────────┘
```

### Determinism Extension — Role-Based Policies

The `aot-determinism.ts` extension reads `PI_STAGE` and `PI_ROLE` env vars to enforce:

| Tool | agent-manage | agent-implement |
|------|-------------|----------------|
| `openspec *` | ALLOW | BLOCK |
| `ask_user` | ALLOW | BLOCK (escalate via tool result) |
| `spawn_agent` | ALLOW | ALLOW (sub-impl only) |
| `write` (repo files) | BLOCK | ALLOW |
| `edit` (repo files) | BLOCK | ALLOW |
| `bash` | BLOCK (except openspec) | ALLOW |
| `read` | ALLOW | ALLOW |

### Subagent Tracking

When an agent spawns a subagent via pi's `subagent/` extension:
1. Extension logs a `subagent_start` event to JSONL with `{parentId, childId, task}`
2. Sidecar captures this and writes to trace spans
3. Structured logs parser emits `subagent` type entries
4. Activity feed renders as nested/indented entries
5. Trace timeline shows parent-child span relationships

### UI Component Plan

Replace/upgrade:
| Current | Action |
|---------|--------|
| Custom toast | Keep (already lightweight) |
| RunStatusBadge | Use shadcn `Badge` with variant |
| Tab bar (raw buttons) | shadcn `Tabs` |
| Info overlay (raw div) | shadcn `Sheet` (slide-over) |
| HITL input overlay | shadcn `Dialog` or `Sheet` |
| File tree | Keep (specialized, no shadcn equiv) |
| TraceTimeline | Keep (specialized flame graph) |
| ActivityFeed | Keep (domain-specific) |
| StageProgress | shadcn `Progress` + `Badge` |
| Command palette | Keep cmdk (already good) |

Rename in UI:
- "unc" → "manage" (label color: blue)
- "neph" → "impl" (label color: green)
- "system" → "system" (label color: yellow)
- "user" → "user" (label color: white/foreground)

### Implementation Strategy

Phase 1 (workspace + naming): Rename unc/neph, fix workspace layout, move openspec to .openspec/
Phase 2 (role separation): Split determinism extension into role-based policies, update system prompts
Phase 3 (persistent manage): Restructure Temporal workflow so manage agent persists across stages
Phase 4 (subagents): Add spawn_agent tool, subagent tracking, UI visibility
Phase 5 (UI polish): shadcn component adoption, consistent theming
