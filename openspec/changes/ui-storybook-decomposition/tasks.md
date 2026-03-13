## 1. Merge UI Branch

- [ ] 1.1 Merge `origin/ui` into `main`, resolving conflicts by taking the `ui` branch's `web/` directory wholesale (keep docs changes too)
- [ ] 1.2 Remove SolidJS-specific files that conflict: `web/src/pages/`, `web/src/components/CreateRunForm.tsx`, `web/src/components/HumanInputForm.tsx`, `web/src/components/EventLog.tsx`, `web/src/components/AgentRunList.tsx`, SolidJS store from `packages/shared`
- [ ] 1.3 Run `npm install` in `web/` to install React + Tailwind deps
- [ ] 1.4 Verify `npm run build` succeeds in `web/`

## 2. Wire Components to Real API

- [ ] 2.1 Add `@connectrpc/connect`, `@connectrpc/connect-web`, `@bufbuild/protobuf` to `web/package.json` (or import AOTClient from packages/shared)
- [ ] 2.2 Create `web/src/hooks/useClient.ts` — React context + hook providing AOTClient instance, configured from `VITE_API_URL` env var
- [ ] 2.3 Replace mock data in `App.tsx` with real API calls: `listAgentRuns()` on mount with polling, `getAgentRun()` for detail, `watchAgentRun()` for streaming
- [ ] 2.4 Wire `AgentRunForm` submit to `AOTClient.createAgentRun()` — map form fields to proto `AgentRunSpec`
- [ ] 2.5 Wire cancel button to `AOTClient.cancelAgentRun()`
- [ ] 2.6 Wire HITL input form to `AOTClient.sendHumanInput()` — show input form when phase is `waiting_for_input`
- [ ] 2.7 Wire `WatchAgentRun` streaming to `EventsView` — append events in real-time via `watchAgentRun()` async iterator

## 3. UX Patterns

- [ ] 3.1 Add skeleton loader component (`web/src/components/Skeleton.tsx`) using Tailwind `animate-pulse`
- [ ] 3.2 Add loading skeletons to AgentRunTable (skeleton rows) and AgentRunDetailPanel (skeleton blocks)
- [ ] 3.3 Add empty states: "No agent runs yet" with Create button in table, "No events yet" in events view
- [ ] 3.4 Add error boundary component (`web/src/components/ErrorBoundary.tsx`) with "Something went wrong" + Retry
- [ ] 3.5 Add toast notification system (`web/src/components/Toast.tsx`) — success/error toasts, auto-dismiss after 5s
- [ ] 3.6 Add toast calls to create, cancel, and send-input actions

## 4. Storybook Setup

- [ ] 4.1 Install Storybook: `npx storybook@latest init --type react` in `web/`, configure with `@storybook/react-vite`
- [ ] 4.2 Configure Storybook to load Tailwind styles (`web/src/index.css` with CSS variables) in `.storybook/preview.ts`
- [ ] 4.3 Add `storybook` and `build-storybook` scripts to `web/package.json`

## 5. Component Stories

- [ ] 5.1 `StatusBadge.stories.tsx` — all 6 phase variants
- [ ] 5.2 `AgentRunTable.stories.tsx` — populated, empty, loading states
- [ ] 5.3 `AgentRunDetailPanel.stories.tsx` — running, waiting_for_input, completed, failed states
- [ ] 5.4 `AgentRunForm.stories.tsx` — default form, validation states
- [ ] 5.5 `Layout.stories.tsx` + `Sidebar.stories.tsx` — shell with sidebar collapsed/expanded
- [ ] 5.6 `ConfirmDialog.stories.tsx` — open with cancel/confirm
- [ ] 5.7 `EventsView.stories.tsx` — with events, empty state
- [ ] 5.8 `Skeleton.stories.tsx` + `ErrorBoundary.stories.tsx` + `Toast.stories.tsx`

## 6. Verification

- [ ] 6.1 `npm run build` in `web/` passes
- [ ] 6.2 `npx tsc --noEmit` in `web/` passes
- [ ] 6.3 `npm run storybook` launches and all stories render correctly
- [ ] 6.4 Verify dev mode works: `npm run dev` shows the app with Vite proxy to API
