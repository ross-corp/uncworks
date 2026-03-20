## 1. Workspace Layout

- [x] 1.1 Update hydration `Run()` to create worktrees at `/workspace/<repo-name>/` instead of `/workspace/src/<repo-name>/`
- [x] 1.2 Update `PrimaryWorktreePath()` to return `/workspace/<repo-name>/`
- [x] 1.3 Update `composeDevbox()` paths to match new layout
- [x] 1.4 Update sidecar `resolveWorkDir()` to detect repos at workspace root (no `/src/` prefix)
- [x] 1.5 Move OpenSpec init/scaffold to `/workspace/.openspec/` (dot-prefix, workspace level)
- [x] 1.6 Update PlanRun/VerifyRun `specDir` from `/workspace` to use `.openspec/` paths
- [x] 1.7 Update hydration tests for new paths

## 2. Rename unc/neph to manage/impl

- [x] 2.1 Rename UI labels in ActivityFeed.tsx: "unc" → "manage", "neph" → "impl"
- [x] 2.2 Rename ThinkingEntry label from "neph" to "impl"
- [x] 2.3 Update sidecar system prompts: references to roles
- [x] 2.4 Update plan prompt builder: role references
- [x] 2.5 Update code comments referencing unc/neph

## 3. Role-Based Determinism

- [x] 3.1 Add `PI_ROLE` env var to agent process (manage or implement)
- [x] 3.2 Update determinism extension: read `PI_ROLE` and enforce tool policies
- [x] 3.3 Manage agent: allow openspec CLI, ask_user, read; block write/edit/bash (except openspec)
- [x] 3.4 Implement agent: allow read/write/edit/bash; block openspec CLI, ask_user
- [x] 3.5 Pass `PI_ROLE` from sidecar StartAgent based on stage

## 4. Persistent Manage Agent

- [ ] 4.1 Restructure spec-driven workflow: manage agent persists across plan+verify stages
- [ ] 4.2 Implement agent spawned by manage during execute stage
- [ ] 4.3 Manage agent monitors implement progress and can steer/interrupt
- [ ] 4.4 On verify failure, manage agent spawns new implement with failure context

## 5. Subagent Support

- [ ] 5.1 Register `spawn_agent` tool in determinism extension (uses pi's subagent API)
- [ ] 5.2 Log subagent lifecycle events to JSONL (subagent_start, subagent_end)
- [ ] 5.3 Parse subagent events in structured logs endpoint
- [ ] 5.4 Display subagent entries in ActivityFeed with indentation
- [ ] 5.5 Show subagent spans in TraceTimeline as child spans

## 6. UI Professionalization

- [x] 6.1 Replace tab bar with shadcn `Tabs` component
- [x] 6.2 Replace info overlay with shadcn `Sheet`
- [x] 6.3 Replace stage progress with shadcn `Progress` + `Badge`
- [x] 6.4 Use shadcn `Badge` variants for RunStatusBadge
- [ ] 6.5 Ensure all themes work with new components
- [ ] 6.6 Verify command palette theme switching applies everywhere
