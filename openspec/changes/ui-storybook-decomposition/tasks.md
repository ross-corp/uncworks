## 1. Merge UI Branch

- [x] 1.1 Merge `origin/ui` into `main`, resolving conflicts by taking the `ui` branch's `web/` directory wholesale (keep docs changes too)
- [x] 1.2 Remove SolidJS-specific files that conflict: `web/src/pages/`, `web/src/components/CreateRunForm.tsx`, `web/src/components/HumanInputForm.tsx`, `web/src/components/EventLog.tsx`, `web/src/components/AgentRunList.tsx`, SolidJS store from `packages/shared`
- [x] 1.3 Run `npm install` in `web/` to install React + Tailwind deps
- [x] 1.4 Verify `npm run build` succeeds in `web/`

## 2. Wire Components to Real API

- [x] 2.1 Add `@connectrpc/connect`, `@connectrpc/connect-web`, `@bufbuild/protobuf` to `web/package.json` (or import AOTClient from packages/shared)
- [x] 2.2 Create `web/src/hooks/useClient.ts` — React context + hook providing AOTClient instance, configured from `VITE_API_URL` env var
- [x] 2.3 Replace mock data in `App.tsx` with real API calls: `listAgentRuns()` on mount with polling, `getAgentRun()` for detail, `watchAgentRun()` for streaming
- [x] 2.4 Wire `AgentRunForm` submit to `AOTClient.createAgentRun()` — map form fields to proto `AgentRunSpec`
- [x] 2.5 Wire cancel button to `AOTClient.cancelAgentRun()`
- [x] 2.6 Wire HITL input form to `AOTClient.sendHumanInput()` — show input form when phase is `waiting_for_input`
- [x] 2.7 Wire `WatchAgentRun` streaming to `EventsView` — append events in real-time via `watchAgentRun()` async iterator

## 3. UX Patterns

- [x] 3.1 Add skeleton loader component (`web/src/components/Skeleton.tsx`) using Tailwind `animate-pulse`
- [x] 3.2 Add loading skeletons to AgentRunTable (skeleton rows) and AgentRunDetailPanel (skeleton blocks)
- [x] 3.3 Add empty states: "No agent runs yet" with Create button in table, "No events yet" in events view
- [x] 3.4 Add error boundary component (`web/src/components/ErrorBoundary.tsx`) with "Something went wrong" + Retry
- [x] 3.5 Add toast notification system (`web/src/components/Toast.tsx`) — success/error toasts, auto-dismiss after 5s
- [x] 3.6 Add toast calls to create, cancel, and send-input actions

## 4. Storybook Setup

- [x] 4.1 Install Storybook: `npx storybook@latest init --type react` in `web/`, configure with `@storybook/react-vite`
- [x] 4.2 Configure Storybook to load Tailwind styles (`web/src/index.css` with CSS variables) in `.storybook/preview.ts`
- [x] 4.3 Add `storybook` and `build-storybook` scripts to `web/package.json`

## 5. Component Stories

- [x] 5.1 `StatusBadge.stories.tsx` — all 6 phase variants
- [x] 5.2 `AgentRunTable.stories.tsx` — populated, empty, loading states
- [x] 5.3 `AgentRunDetailPanel.stories.tsx` — running, waiting_for_input, completed, failed states
- [x] 5.4 `AgentRunForm.stories.tsx` — default form, validation states
- [x] 5.5 `Layout.stories.tsx` + `Sidebar.stories.tsx` — shell with sidebar collapsed/expanded
- [x] 5.6 `ConfirmDialog.stories.tsx` — open with cancel/confirm
- [x] 5.7 `EventsView.stories.tsx` — with events, empty state
- [x] 5.8 `Skeleton.stories.tsx` + `ErrorBoundary.stories.tsx` + `Toast.stories.tsx`

## 6. Verification

- [x] 6.1 `npm run build` in `web/` passes
- [x] 6.2 `npx tsc --noEmit` in `web/` passes
- [x] 6.3 `npm run build-storybook` completes successfully (static build verified)
- [x] 6.4 Verify dev mode works: Vite build succeeds, storybook build succeeds
