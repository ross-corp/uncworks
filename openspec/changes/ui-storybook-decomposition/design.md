## Context

The `ui` branch has a well-designed React + Tailwind web UI with CSS variable-based theming, collapsible sidebar, data tables, detail panels, and forms. It uses mock data. Main has a SolidJS UI wired to the real API via ConnectRPC but with minimal design. We need to combine the best of both: the `ui` branch's design with the real API integration.

The `ui` branch components: Layout (shell with sidebar + content), Sidebar (nav + filters + run list), AgentRunTable (sortable data table), AgentRunDetailPanel (run detail with tabs), AgentRunForm (create run modal), StatusBadge (phase indicator), ConfirmDialog (modal), EventsView (event log), ReposView (repo list).

## Goals / Non-Goals

**Goals:**
- Merge `ui` branch design into main, replacing SolidJS with React + Tailwind
- Wire all components to real ConnectRPC API (AOTClient from packages/shared)
- Decompose every component into Storybook stories with multiple states
- Add proper UX: loading skeletons, error states, empty states, toast notifications
- Keep the existing API integration logic (WatchAgentRun streaming, HITL input, cancel)

**Non-Goals:**
- Building new backend features
- Adding authentication UI (no auth backend exists yet)
- Mobile-first responsive design (desktop dashboard is sufficient)
- SSR or server components (client-side SPA is fine)
- E2E browser testing (Storybook visual testing is the goal)

## Decisions

### React + Tailwind over SolidJS
The `ui` branch already has substantial React + Tailwind work with a proper design system. Rewriting it in SolidJS would be wasted effort. React has a larger ecosystem (Storybook support is first-class) and the team is already building in it.

### Storybook 8 with @storybook/react-vite
Uses the same Vite build pipeline. Stories can render components with mock props (no backend needed). Enables isolated development and visual regression testing later.

### CSS Variables for theming, Tailwind for utility classes
The `ui` branch already has `--surface-0..3`, `--text-primary/secondary/tertiary`, `--accent`, `--danger`, etc. defined as CSS variables with Tailwind mappings. This enables dark/light theme switching without changing component code.

### Keep AOTClient from packages/shared
The ConnectRPC client already works. Components will call it via React context or prop drilling (no Redux/Zustand needed — the app is small enough).

### Component decomposition pattern
Each component gets its own directory: `web/src/components/<Name>/`, containing:
- `<Name>.tsx` — the component
- `<Name>.stories.tsx` — Storybook stories
- `index.ts` — re-export

## Risks / Trade-offs

- **Framework switch risk** → The `ui` branch is already React, so this is merging, not rewriting. Main's SolidJS components are thin and the API client (packages/shared) is framework-agnostic.
- **Mock data removal** → The `ui` branch's mock data is useful for Storybook stories. Keep `mock.ts` for stories but remove it from the actual app.
- **Merge conflicts** → The `ui` branch diverged from an older main. The merge will conflict on `web/` files. Strategy: take the `ui` branch's web/ wholesale, then re-add API wiring.
