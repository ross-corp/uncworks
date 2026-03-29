## Context

The app has a full set of list views (Runs, Projects, Templates, Chains, Schedules, ChainRuns) and a New Run form. Current gaps:
- Loading states use bare text "Loading..." with no visual indicator
- Empty states use bare centered text with no icon or CTA in 3 of 6 views
- `NewRunView` has no warning when the LLM key is not configured, causing silent failures
- `ProjectListView` and `ScheduleListView` silently swallow fetch errors
- `TemplateListView` has duplicate fetch logic (inline `useEffect` + `fetchData` callback both hit the same endpoint)

Existing components available: `Spinner` (`components/ui/spinner.tsx`), `Empty` / `EmptyHeader` / `EmptyTitle` / `EmptyDescription` / `EmptyContent` (`components/ui/empty.tsx`), `Alert` / `AlertDescription` (`components/ui/alert.tsx`). The `useSettings()` hook from `hooks/useSettings.tsx` exposes `configStatus.hasLLMKey`.

## Goals / Non-Goals

**Goals:**
- Consistent loading state (Spinner) in all list views
- Consistent empty states (Empty + CTA) in all list views
- Config-gate banner on NewRunView when LLM key is absent
- Error toasts on ProjectListView and ScheduleListView
- Remove duplicate fetch in TemplateListView

**Non-Goals:**
- Redesigning list row layouts
- Changes to RunListView (already has loading/empty states)
- Changing Settings layout or padding
- Changing GlobalNav
- Any API changes

## Decisions

**D1: Use existing `Spinner` and `Empty` components, not new ones.**
Both components exist and are unused in list views. Using them keeps the pattern consistent with the rest of the design system. Alternative (bare text) is current broken state.

**D2: Config gate on NewRunView uses an inline Alert, not a modal or disabled button.**
An inline amber Alert below the header is non-blocking — it informs but lets the user proceed (useful when they know the key is set server-side but the Wails settings haven't loaded yet). Disabling the submit button entirely would be too aggressive and confusing if settings haven't loaded.

**D3: Remove duplicate fetch in TemplateListView by keeping only the `useEffect` pattern.**
The `fetchData` callback + `useEffect` + interval pattern is duplicated. Keep the inline `useEffect` with cancel + interval (the same pattern used in ChainListView/ScheduleListView), and use the callback only for imperative refreshes after delete.

**D4: Error handling uses `toast.error` for consistency.**
RunListView, TemplateListView, ChainListView all use `toast.error`. ProjectListView and ScheduleListView silently swallow errors. Fix them to match.

## Risks / Trade-offs

- [Risk] Config banner in NewRunView may flash briefly on initial load before settings resolve → Mitigation: check `loading` from `useSettings()` and hide the banner while loading is true.
- [Risk] TemplateListView duplicate fetch removal could break imperative refresh after delete → Mitigation: the `fetchData` callback is kept for the delete handler; only the duplicate `useEffect` polling block is removed.

## Migration Plan

No migrations needed. These are pure UI changes in `.tsx` files. TypeScript check (`npx tsc --noEmit`) validates correctness before commit.
