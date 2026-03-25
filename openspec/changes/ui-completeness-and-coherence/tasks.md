## 1. Routes and Navigation

- [x] 1.1 Add `/templates` route in AppNew.tsx pointing to new TemplateListView
- [x] 1.2 Add `/templates/new` route in AppNew.tsx pointing to new TemplateNewView
- [x] 1.3 Add `/chains/new` route in AppNew.tsx pointing to new ChainNewView
- [x] 1.4 Add `/schedules/new` route in AppNew.tsx pointing to new ScheduleNewView
- [x] 1.5 Add Templates nav item to GlobalNav.tsx (icon `◻`, path `/templates`, countKey `templates`) — insert between Projects and Chains
- [x] 1.6 Change Chain Runs icon in GlobalNav from `⛓` to `↩` to distinguish from Chains
- [x] 1.7 Add `templates` count to GlobalNav fetchCounts (GET /api/v1/templates, total length)

## 2. Template Management

- [x] 2.1 Create `web/src/views/TemplateListView.tsx` — list all templates, show name/displayName/description/runCount/age, "+ new template" button, delete button per row with 409 handling via toast.error
- [x] 2.2 Create `web/src/views/TemplateNewView.tsx` — form with fields: name (slug input, required), displayName, description, projectRef (select from GET /api/v1/projects), prompt (textarea); POST /api/v1/templates on submit; redirect to /templates on success; toast.success/toast.error

## 3. Chain Create

- [x] 3.1 Create `web/src/views/ChainNewView.tsx` — form with: name (required), displayName, description, projectRef select (from GET /api/v1/projects); steps builder section; submit button disabled when name empty or no steps
- [x] 3.2 Implement steps builder in ChainNewView: "+ add step" button appends a step row with: step name input, templateRef select (from GET /api/v1/templates), dependsOn multi-select (checkboxes from prior step names), remove button
- [x] 3.3 Wire ChainNewView submit: POST /api/v1/chains with `{name, spec: {displayName, description, projectRef, steps}}`; redirect to /chains on success; toast.success/toast.error on failure
- [x] 3.4 Add delete button per chain row in ChainListView.tsx: call DELETE /api/v1/chains/:name; refresh list; toast.error on 409 showing API message
- [x] 3.5 Add "+ new chain" button to ChainListView.tsx header linking to /chains/new

## 4. Schedule Create

- [x] 4.1 Create `web/src/views/ScheduleNewView.tsx` — form with: name (required), displayName, cron (required, with cronstrue live preview below), timezone (text input, default "UTC"), concurrencyPolicy select (Allow/Forbid/Replace, default Forbid), suspend checkbox
- [x] 4.2 Add target type radio toggle (Chain / Template) in ScheduleNewView: when Chain selected show select from GET /api/v1/chains; when Template selected show select from GET /api/v1/templates
- [x] 4.3 Wire ScheduleNewView submit: POST /api/v1/schedules; redirect to /schedules on success; toast.success/toast.error on failure; submit disabled when name empty, cron invalid, or no target selected
- [x] 4.4 Add delete button per schedule row in ScheduleListView.tsx: call DELETE /api/v1/schedules/:name; refresh list; toast.success/toast.error
- [x] 4.5 Replace "+ new schedule" link in ScheduleListView.tsx header and remove the redundant "Runs" and "Chains" nav buttons from the header

## 5. Project Settings Expansion

- [x] 5.1 Add devbox packages section to ProjectDetailView.tsx Settings tab: list current devbox.packages strings; text input + "+" button to add; remove button per package; marks settingsDirty on change
- [x] 5.2 Add "Run Defaults" section to ProjectDetailView.tsx Settings tab: modelTier select (""/"economy"/"standard"/"performance"), manageModelTier select (same options), implementModelTier select (same options), ttlSeconds number input, orchestrationMode select (""/"spec-driven"/"prompt-driven"), autoPush checkbox, autoPR checkbox, prBaseBranch text input (disabled when autoPR off)
- [x] 5.3 Update saveSettings in ProjectDetailView.tsx to include devbox and defaults in the PUT /api/v1/projects/:name body
- [x] 5.4 Update fetchProject in ProjectDetailView.tsx to populate devbox and defaults state from API response

## 6. Design Coherence Sweep

- [x] 6.1 RunListView.tsx — replace all custom h-* Button classes with `size="sm"`; header to h-12 flex items-center; rows to px-4 py-2.5; remove text-[10px]/text-[11px]
- [x] 6.2 NewRunView.tsx — replace `useToast` import with sonner `toast`; convert all `toast(msg, "error")` → `toast.error(msg)` and `toast(msg, "success")` → `toast.success(msg)`; apply button size="sm"; fix any h-* custom sizes
- [x] 6.3 RunDetailView.tsx — apply header h-12; button size="sm"; remove text-[10px]/text-[11px]; verify three-zone layout doesn't overflow with GlobalNav (check flex container widths)
- [x] 6.4 ProjectListView.tsx — header h-12; rows px-4 py-2.5; button size="sm"; remove micro text sizes
- [x] 6.5 ProjectDetailView.tsx — header h-12; button size="sm"; remove text-[10px]/text-[11px]
- [x] 6.6 ChainListView.tsx — header h-12; rows px-4 py-2.5; button size="sm"; remove text-[10px]/text-[11px]
- [x] 6.7 ChainRunListView.tsx — header h-12; rows px-4 py-2.5; button size="sm"
- [x] 6.8 ChainRunDetailView.tsx — header h-12; button size="sm"; remove micro text sizes
- [x] 6.9 ScheduleListView.tsx — header h-12; rows px-4 py-2.5; button size="sm"; remove text-[10px]/text-[11px]
- [x] 6.10 ScheduleDetailView.tsx — header h-12; button size="sm"; remove micro text sizes
- [x] 6.11 TraceTimeline.tsx — remove text-[10px]/text-[11px]; button size="sm" for filter chips; keep functional behavior unchanged

## 7. Verification

- [x] 7.1 `tsc --noEmit` passes with no errors
- [x] 7.2 `vite build` succeeds
- [ ] 7.3 Manually verify: create a RunTemplate → shows in /templates list
- [ ] 7.4 Manually verify: create a Chain using that template → shows in /chains with DAG steps
- [ ] 7.5 Manually verify: create a Schedule pointing to the chain → shows in /schedules
- [ ] 7.6 Manually verify: Project Settings shows devbox packages and defaults sections; save updates CRD
- [ ] 7.7 Manually verify: GlobalNav shows Templates between Projects and Chains; counts update
- [ ] 7.8 Manually verify: ScheduleListView has no "Runs"/"Chains" nav buttons
- [ ] 7.9 Manually verify: no text-[10px] or text-[11px] in rendered HTML (browser devtools)
