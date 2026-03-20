## Why

The platform currently uses a single agent process (pi) for all pipeline stages (plan/execute/verify), with informal "unc" and "neph" labels in the UI. The agent writes directly to the repo's own `openspec/` directory, repos are nested under `/workspace/src/` instead of being workspace-root worktrees, and there's no formal separation between the managing/planning role and the implementing role. This makes the system brittle, non-deterministic, and hard to reason about. We need a clean architectural split with deterministic policy enforcement, proper workspace layout, subagent support, and a professional UI.

## What Changes

- **BREAKING** Rename "unc" to "agent-manage" and "neph" to "agent-implement" throughout codebase (UI labels, system prompts, activity feed, code comments)
- Establish agent-manage as a persistent orchestrator that stays alive across pipeline stages, supervising agent-implement
- agent-manage owns specs/planning (runs openspec CLI), agent-implement owns code (writes to repo)
- Separate contexts: each agent role gets distinct tools, permissions, and system prompts enforced by the determinism extension
- Repos cloned as git worktrees directly into `/workspace/` root (no `/workspace/src/` prefix)
- OpenSpec artifacts live at `/workspace/.openspec/` (workspace-level, not inside repo)
- Subagent spawning: both agents can spawn subagents, with visibility in the UI activity feed and trace timeline
- Replace custom UI components with shadcn equivalents where possible (tabs, cards, dialogs, toasts)
- Determinism extension enforces role-specific tool policies per agent type

## Capabilities

### New Capabilities
- `agent-role-separation`: Distinct agent-manage and agent-implement roles with separate contexts, tools, and permissions
- `workspace-layout`: Repos as workspace-root worktrees, openspec at workspace level
- `subagent-visibility`: UI displays subagent trees in activity feed and traces
- `deterministic-policy`: Role-based tool restrictions enforced by pi extension
- `ui-professionalization`: shadcn component adoption, consistent theming, professional layout

### Modified Capabilities

(none — these are all new capabilities layered on existing infrastructure)

## Impact

- **Hydration init container**: Change worktree target from `/workspace/src/<repo>` to `/workspace/<repo>` (or `/workspace/` for single repo)
- **Sidecar gateway**: resolveWorkDir logic, system prompts, agent process management for concurrent agents
- **Temporal workflows**: Workflow restructure for persistent manage agent with delegated implement runs
- **Determinism extension**: Role-aware policy enforcement, subagent spawn tracking
- **Web UI**: All components referencing unc/neph, ActivityFeed, TraceTimeline, RunDetailView, Layout
- **Helm chart**: No changes needed (agent images are the same binary)
- **API/Proto**: May need subagent status fields in AgentRun CRD
