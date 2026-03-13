## Why

The web UI exists in two diverged states: `main` has a SolidJS skeleton wired to the real API but with minimal UX thought, while the `ui` branch has a polished React + Tailwind design system with proper layout, sidebar, tables, and forms — but uses mock data. We need to merge the `ui` branch's design work into `main`, switch the framework to React + Tailwind, wire it to the real ConnectRPC API, and decompose every component for Storybook so we can iterate on UX independently of backend concerns.

## What Changes

- Merge `origin/ui` branch into `main`, replacing the current SolidJS web UI with the React + Tailwind implementation
- Add Storybook with stories for every UI component (Layout, Sidebar, AgentRunTable, AgentRunDetailPanel, AgentRunForm, StatusBadge, ConfirmDialog, EventsView, ReposView)
- Replace mock data layer with the real ConnectRPC `AOTClient` from `packages/shared`
- Add proper UX patterns: loading states, error boundaries, empty states, skeleton loaders, toast notifications
- Ensure all existing functionality works: create run, list runs, run detail with streaming events, HITL input, cancel

## Capabilities

### New Capabilities
- `storybook-component-library`: Storybook setup with stories for all UI components, demonstrating each in isolation with various states (loading, error, empty, populated)
- `ui-ux-patterns`: Loading skeletons, error boundaries, toast notifications, empty states, and responsive layout handling

### Modified Capabilities

## Impact

- `web/` — complete rewrite: SolidJS → React, inline styles → Tailwind, add Storybook
- `web/package.json` — new deps: react, tailwindcss, storybook, @storybook/react-vite
- `packages/shared/` — AOTClient stays the same, but import paths change (SolidJS reactive store removed)
- Removes: `@solidjs/router`, `solid-js`, SolidJS-specific components and store
- Adds: `react`, `react-dom`, `tailwindcss`, `@storybook/react-vite`, `postcss`, `autoprefixer`
