## 1. Loading States

- [x] 1.1 Replace "Loading..." text in `ProjectListView` with centered `Spinner`
- [x] 1.2 Replace "Loading..." text in `TemplateListView` with centered `Spinner`
- [x] 1.3 Replace "Loading..." text in `ChainListView` with centered `Spinner`
- [x] 1.4 Replace "Loading..." text in `ScheduleListView` with centered `Spinner`
- [x] 1.5 Replace "Loading..." text in `ChainRunListView` with centered `Spinner`

## 2. Empty States

- [x] 2.1 Upgrade `ProjectListView` empty state to use `Empty` / `EmptyHeader` / `EmptyTitle` / `EmptyDescription` with "No projects yet" and a "+ new project" CTA button
- [x] 2.2 Upgrade `ChainListView` empty state to use `Empty` components with "No chains defined" and a "+ new chain" CTA
- [x] 2.3 Upgrade `ScheduleListView` empty state to use `Empty` components with "No schedules configured" and a "+ new schedule" CTA
- [x] 2.4 Upgrade `ChainRunListView` empty state to use `Empty` components with "No chain runs yet" and a "View Chains" CTA
- [x] 2.5 Upgrade `TemplateListView` empty state to use `Empty` wrapper for visual consistency (had CTA already; upgraded to structured layout)

## 3. Error Handling

- [x] 3.1 Add `toast.error` in `ProjectListView.fetchProjects` catch block (was silent)
- [x] 3.2 Add `toast.error` in `ScheduleListView.fetchData` catch block (was silent)

## 4. Duplicate Fetch Cleanup

- [x] 4.1 Remove the duplicate inline `useEffect` fetch loop in `TemplateListView` (kept `fetchData` callback + single interval-based `useEffect`)

## 5. Config Gate on NewRunView

- [x] 5.1 Import `useSettings` into `NewRunView`
- [x] 5.2 Add an amber warning banner below the header when `!configStatus.hasLLMKey && !settingsLoading`, linking to `/settings`

## 6. TypeScript Validation

- [x] 6.1 Run `cd web && npx tsc --noEmit` — zero errors
