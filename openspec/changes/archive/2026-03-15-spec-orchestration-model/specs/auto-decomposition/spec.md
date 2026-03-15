## ADDED Requirements

### Requirement: Senior agent produces structured decomposition plan
When `orchestrationMode` is `auto`, the workflow SHALL start a senior agent with an augmented prompt containing the original spec/prompt plus decomposition instructions. The senior agent SHALL produce a JSON decomposition plan with a `tasks` array (max 7 items) and an `integration_prompt`.

#### Scenario: Spec with multiple independent concerns
- **WHEN** an AgentRun with `orchestrationMode=auto` starts
- **THEN** the workflow starts the agent with a decomposition preamble prompt
- **AND** the agent produces a JSON object containing `tasks` (array of `{name, prompt, repos}`) and `integration_prompt`
- **AND** the workflow parses the JSON and spawns one junior AgentRun per task

#### Scenario: Spec is simple enough for one agent
- **WHEN** the senior agent determines the spec needs no decomposition
- **THEN** it returns `{"tasks": []}`
- **AND** the workflow falls back to single-run execution with the original prompt

#### Scenario: Senior produces malformed JSON
- **WHEN** the senior agent's output cannot be parsed as valid JSON
- **THEN** the workflow logs a warning with the raw output
- **AND** the workflow falls back to single-run execution with the original prompt

### Requirement: Juniors execute in parallel
The workflow SHALL spawn all junior AgentRuns concurrently using Temporal child workflows. The senior workflow SHALL wait for all juniors to reach a terminal state before proceeding to integration.

#### Scenario: All juniors succeed
- **WHEN** all junior AgentRuns reach phase `Succeeded`
- **THEN** the senior workflow proceeds to the integration step

#### Scenario: Some juniors fail
- **WHEN** one or more junior AgentRuns reach phase `Failed`
- **THEN** the senior workflow still proceeds to integration
- **AND** the integration prompt includes the failure messages for failed juniors

#### Scenario: Junior is cancelled
- **WHEN** the senior AgentRun is cancelled while juniors are running
- **THEN** all junior AgentRun workflows receive cancel signals
- **AND** the senior transitions to `Cancelled`

### Requirement: Senior integrates junior results
After all juniors complete, the workflow SHALL collect each junior's git diff output and start the senior agent with a review prompt containing the original spec, each junior's task description, status, and diff. The senior reviews, resolves conflicts, and produces a final integration.

#### Scenario: Integration with clean diffs
- **WHEN** all juniors succeeded and their diffs do not conflict
- **THEN** the senior applies all diffs and verifies the result

#### Scenario: Integration with conflicting diffs
- **WHEN** multiple juniors modified the same files
- **THEN** the senior's review prompt flags the overlapping files
- **AND** the senior resolves conflicts in its workspace

### Requirement: Decomposition capped at 7 tasks
The decomposition preamble SHALL instruct the senior to produce at most 7 tasks. If the workflow receives more than 7 tasks in the JSON output, it SHALL truncate to the first 7 and log a warning.

#### Scenario: Senior produces 10 tasks
- **WHEN** the senior's JSON output contains 10 tasks
- **THEN** the workflow uses only the first 7
- **AND** a warning is logged indicating truncation
