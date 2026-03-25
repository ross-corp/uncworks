## Context

The ui-overhaul-v2 was implemented by 6 parallel agents, each owning non-overlapping files. This avoided git conflicts but produced divergent micro-decisions: 4 button height conventions, 4 text size scales, 2 toast systems, and header heights that differ by view. Simultaneously, CRUD creation flows for Chains, Schedules, and RunTemplates were never built — those entities only appear in list views with no way to create them. The backend APIs all exist; only the frontend is missing.

## Goals / Non-Goals

**Goals:**
- Every entity (RunTemplate, Chain, Schedule, Project) fully creatable and deletable from UI
- One consistent visual language: single button scale, header height, row padding, text scale, toast system
- Project Settings surfaces all editable fields the API accepts
- Zero redundant navigation elements

**Non-Goals:**
- Editing existing chains or schedules in-place (read + delete is sufficient for now)
- RunTemplate editing (create + delete only)
- Mobile/responsive layout
- Backend API changes

## Decisions

**D1: Create forms as full routes, not dialogs**
Chains require a multi-step DAG builder that needs significant screen space. Using `/chains/new` and `/schedules/new` routes (rather than dialogs) is consistent with `/new` for runs and avoids cramming complex forms into modals. Templates are simpler but follow the same pattern for consistency.

**D2: Design tokens via Tailwind class conventions, not a separate token file**
Since the codebase is Tailwind-only with shadcn/ui, the "design system" is a documented class convention enforced by code review. No new abstraction layer needed. Conventions: `size="sm"` for all buttons, `px-4 py-2.5` for rows, `h-12` for headers, `text-sm`/`text-xs` only.

**D3: Standardize on sonner for all toasts**
`useToast` (custom hook) is only used in NewRunView. Sonner is already the standard everywhere else and is the shadcn-recommended pattern. Remove the custom hook dependency from NewRunView.

**D4: RunTemplates in GlobalNav**
Templates are a first-class entity (chains can't be built without them). They belong in the nav between Projects and Chains. Icon: `◻` (distinct from chain `⛓`). Count badge shows total templates.

**D5: Chain step builder is additive, linear UI**
Rather than a drag-and-drop DAG editor, steps are added sequentially with a `dependsOn` multi-select from existing step names. The read-side DAG viz (ChainDagViz) already exists — the create form just needs to produce valid JSON. Simplicity wins over visual polish for an admin-facing form.

**D6: Schedule target is a toggle (Chain vs Template)**
`chainRef` and `templateRef` are mutually exclusive in the CRD. The form shows a radio toggle (Chain / Template) then a select for the chosen type. Avoids confusing dual-optional fields.

## Risks / Trade-offs

[Design sweep scope] Touching every view creates a large diff → Mitigation: mechanical changes only (class string substitutions), no logic changes. Each view is self-contained so failures are isolated.

[Template API shape] The `handleCreateTemplate` request body shape isn't shown in the grep output → Mitigation: read `internal/server/chains.go` before implementing the template create form to confirm field names.

[GlobalNav poll count] Adding Templates adds a 6th parallel API call every 10s → Mitigation: acceptable at current scale; counts are fire-and-forget with allSettled.
