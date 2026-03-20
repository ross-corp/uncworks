## Why

NewRunView hardcodes the model tier to "default" when creating agent runs. The `types.go` file defines 7 model tiers (budget, default, balanced, performance, max, custom, router) but the UI never exposes them. Users have no way to choose a cheaper or more powerful model for their run without editing code.

## What Changes

- **Add model tier dropdown**: Add a shadcn Select component to NewRunView that lets users pick a model tier.
- **Populate from types**: Use the `MODEL_TIER_OPTIONS` constant from `types/agent-run.ts` as the dropdown options.
- **Wire into request**: Pass the selected model tier into the `CreateAgentRun` request payload.
- **Cost hints**: Show a brief cost hint next to each option (e.g., "budget - lowest cost", "max - highest quality") if available.

## Capabilities

### New Capabilities

- Users can select a model tier when creating a new agent run from the web UI.

### Modified Capabilities

- The new run form includes an additional field (model tier) that defaults to "default".

## Impact

- `web/src/components/NewRunView.tsx` — Add model tier Select dropdown
- `web/src/types/agent-run.ts` — Ensure `MODEL_TIER_OPTIONS` is exported (may already be)
