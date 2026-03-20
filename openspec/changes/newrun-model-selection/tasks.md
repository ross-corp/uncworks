## 1. Add Model Tier Dropdown

- [x] 1.1 Add shadcn Select component to NewRunView for model tier selection
- [x] 1.2 Populate dropdown options from `MODEL_TIER_OPTIONS` in `types/agent-run.ts`
- [x] 1.3 Set default value to "default"
- [x] 1.4 Add cost/quality hint labels next to each option

## 2. Wire Into Request

- [x] 2.1 Wire selected model tier value into the `CreateAgentRun` request payload
- [x] 2.2 Verify the request includes the selected tier (inspect network tab or add test)

## 3. Verify

- [x] 3.1 Verify TypeScript compiles (`tsc --noEmit`)
- [x] 3.2 Manually test: create runs with different tiers, confirm correct tier in API request
