## Why

After the ui-overhaul-v2, the UI is visually improved but functionally incomplete: core CRUD operations for chains, schedules, and templates are missing entirely, and parallel agent implementation created inconsistent design patterns across every view. Users cannot create chains or schedules from the UI, project settings hide half their configurable fields, and the visual language is incoherent enough to erode trust in the tool.

## What Changes

- **RunTemplates surface**: new list view at `/templates`, create form, delete with 409 guard — currently invisible in UI despite being a core entity
- **Create Chain form**: `/chains/new` with step builder (pick template, set dependsOn DAG edges) and DAG preview
- **Create/Delete Schedule form**: `/schedules/new` with cron editor, human-readable preview, chainRef or templateRef selector
- **Delete actions**: chain and schedule deletion with 409 conflict handling
- **Project Settings expansion**: devbox packages list editor and full defaults section (model tiers, TTL, autoPush/PR, orchestration mode)
- **Design coherence sweep**: standardize button scale, header height, row padding, text sizes, and toast system across all views
- **Navigation cleanup**: remove redundant nav buttons inside list view headers, fix GlobalNav to include Templates

## Capabilities

### New Capabilities
- `template-management`: CRUD UI for RunTemplates (list, create, delete)
- `chain-create`: Form-based chain creation with step builder and DAG dependency wiring
- `schedule-create`: Form-based schedule creation with cron editor and target selector
- `project-settings-full`: Full project settings editing including devbox packages and run defaults
- `ui-design-system`: Enforced design token conventions (button scale, header, row, text, toast)

### Modified Capabilities
- `ui-views`: Navigation structure updated (Templates added, redundant header buttons removed)

## Impact

- `web/src/views/`: All existing views touched for design sweep; new views for templates and create forms
- `web/src/components/GlobalNav.tsx`: New Templates nav item
- `web/src/AppNew.tsx`: New routes: `/templates`, `/chains/new`, `/schedules/new`
- No backend changes — all APIs already exist (`POST /api/v1/chains`, `POST /api/v1/schedules`, `GET/POST /api/v1/templates`, `PUT /api/v1/projects/:name`)
- Removes `web/src/hooks/useToast.ts` dependency from NewRunView (standardize on sonner)
