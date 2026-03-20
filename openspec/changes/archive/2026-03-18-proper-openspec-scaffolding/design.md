## Context

The planning agent writes flat files instead of using `openspec new change`. The system prompt tells it to use the CLI but the agent ignores it — 8B models aren't reliable at following multi-step CLI instructions in system prompts. The solution: scaffold the change programmatically via ExecCommand, then give the agent the exact file paths and format templates.

## Goals / Non-Goals

**Goals:**
- PlanRun creates a valid OpenSpec change before the agent starts
- Agent receives exact file paths and format templates in its prompt
- `openspec validate` passes on the agent's output
- Agent output is always in proper OpenSpec format

**Non-Goals:**
- Teaching the agent to use OpenSpec CLI (unreliable with small models)
- Changing the OpenSpec schema or format
- Adding new OpenSpec features

## Decisions

### Decision 1: Scaffold via ExecCommand, not agent

PlanRun does this before starting the planning agent:
```
1. openspec init --tools pi --force (if needed)
2. openspec new change "<run-id>"
3. openspec instructions proposal --change "<run-id>" --json → get template
4. openspec instructions specs --change "<run-id>" --json → get template
5. openspec instructions tasks --change "<run-id>" --json → get template
```

Then builds the agent prompt with exact paths and templates:
```
Write the proposal to openspec/changes/<id>/proposal.md using this template:
<template content>

Write specs to openspec/changes/<id>/specs/<capability>/spec.md using this format:
<WHEN/THEN template>

Write tasks to openspec/changes/<id>/tasks.md using this format:
<checkbox template>
```

**Rationale:** The agent is good at generating content in a given format. It's bad at running multi-step CLI commands. Give it the structure, let it fill in the content.

### Decision 2: Use openspec instructions for templates

`openspec instructions <artifact> --change <id> --json` returns the exact template and schema guidance for each artifact. We parse this and include it in the agent prompt. This way the agent always writes in the correct OpenSpec format, even if the schema changes in the future.

### Decision 3: No pi extension installation needed

We don't need the agent to use OpenSpec skills — the Temporal activity handles all CLI interactions. The agent just writes markdown files to specific paths. This is simpler and more reliable than trying to get the agent to use skills.

## Risks / Trade-offs

- **Larger prompts** — including templates in the prompt adds ~500 tokens. Acceptable for planning quality.
- **Schema coupling** — the prompt references OpenSpec format. If the format changes, `openspec instructions` output changes automatically, so the prompt stays correct.
