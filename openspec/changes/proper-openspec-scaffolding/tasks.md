## 1. PlanRun: Scaffold Change Before Agent

- [ ] 1.1 In PlanRun, after openspec init, run `openspec new change "<run-id>"` via ExecCommand
- [ ] 1.2 Verify the change was created: `openspec status --change "<run-id>" --json`
- [ ] 1.3 Parse status response to get artifact list and output paths
- [ ] 1.4 If scaffolding fails, return error (don't start the agent)

## 2. PlanRun: Get Templates from openspec instructions

- [ ] 2.1 Run `openspec instructions proposal --change "<run-id>" --json` via ExecCommand
- [ ] 2.2 Parse the instructions response to extract the template field
- [ ] 2.3 Run `openspec instructions specs --change "<run-id>" --json` via ExecCommand
- [ ] 2.4 Parse to extract specs template (WHEN/THEN format)
- [ ] 2.5 Run `openspec instructions tasks --change "<run-id>" --json` via ExecCommand
- [ ] 2.6 Parse to extract tasks template (checkbox format)
- [ ] 2.7 Create `parseOpenSpecInstructionsResponse` parser in openspec_parsers.go

## 3. PlanRun: Build Structured Agent Prompt

- [ ] 3.1 Build prompt that includes exact file paths from status output
- [ ] 3.2 Include proposal template with "write to this path" instruction
- [ ] 3.3 Include specs template with WHEN/THEN format example
- [ ] 3.4 Include tasks template with checkbox format example
- [ ] 3.5 Include the user's original prompt/spec as the "what to plan" content
- [ ] 3.6 Remove the old vague "Create an OpenSpec change" prompt

## 4. Update Plan Stage System Prompt

- [ ] 4.1 Update stageSystemPrompt("plan") in gateway.go to not reference CLI commands
- [ ] 4.2 New prompt: "You are a planning agent. Write structured specs to the file paths provided in your prompt. Follow the templates exactly."
- [ ] 4.3 Remove the 5-step CLI instruction list (scaffolding is done by Temporal now)

## 5. Testing

- [ ] 5.1 Unit test: parseOpenSpecInstructionsResponse with valid/invalid JSON
- [ ] 5.2 Integration test: PlanRun scaffolds change, agent writes files, validate passes
- [ ] 5.3 Verify openspec validate --json passes on agent-written artifacts
