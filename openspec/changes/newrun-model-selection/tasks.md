## 1. Add Model Tier Dropdown

- [ ] 1.1 Add shadcn Select component to NewRunView for model tier selection
- [ ] 1.2 Populate dropdown options from `MODEL_TIER_OPTIONS` in `types/agent-run.ts`
- [ ] 1.3 Set default value to "default"
- [ ] 1.4 Add cost/quality hint labels next to each option

## 2. Wire Into Request

- [ ] 2.1 Wire selected model tier value into the `CreateAgentRun` request payload
- [ ] 2.2 Verify the request includes the selected tier (inspect network tab or add test)

## 3. Verify

- [ ] 3.1 Verify TypeScript compiles (`tsc --noEmit`)
- [ ] 3.2 Manually test: create runs with different tiers, confirm correct tier in API request
