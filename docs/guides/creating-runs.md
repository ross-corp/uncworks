# Creating Runs

Agent runs are the core unit of work in UNCWORKS. Each run clones one or more repositories into a sandboxed workspace, starts an LLM-powered coding agent, and tracks execution through completion.

## Via the Web UI

1. Open the dashboard (default: `http://<host-ip>:30300`)
2. Navigate to the new run page
3. Fill in the required fields:
   - **Repository URL** -- The git repo to clone (e.g., `https://github.com/org/repo`)
   - **Branch** -- Optional; defaults to the repository's default branch
   - **Prompt** -- The task description for the agent (e.g., "Add input validation to the user registration endpoint")
   - **Model** -- Select from available models configured in LiteLLM
4. Choose an orchestration mode:
   - **Single** -- One agent handles the entire task (default)
   - **Auto** -- The agent autonomously decomposes the task into subtasks
   - **Manual** -- You define explicit subtasks with individual prompts
   - **Spec-Driven** -- Uses the OpenSpec Plan/Execute/Verify pipeline
5. Submit the run

## Orchestration Modes

### Single

The simplest mode. A single agent pod receives the prompt and works until completion or failure. Best for focused, well-scoped tasks.

### Auto

The agent receives a decomposition prompt and can use the `delegate_task` tool to break work into tracked subtasks. Subtasks execute inline but are visible in the dashboard.

### Manual

You define an explicit list of subtasks, each with its own prompt and optional repo restriction. Each subtask runs as a separate agent. Useful when you know the exact breakdown.

### Spec-Driven

The most structured mode. Uses OpenSpec to generate formal specifications, then implements and verifies against them. See the [Spec-Driven Pipeline guide](spec-driven.md) for details.

## Monitoring a Run

Once submitted, the run progresses through phases:

| Phase | Description |
|-------|-------------|
| Pending | Run created, waiting for pod provisioning |
| Running | Agent is actively working |
| WaitingForInput | Agent paused, awaiting human input via `ask_user` |
| Succeeded | Agent completed the task |
| Failed | Agent encountered an unrecoverable error |
| Cancelled | User cancelled the run |

The dashboard shows real-time logs, tool calls, trace timelines, and file diffs for each run.

## Human-in-the-Loop

When an agent calls `ask_user`, the run enters the `WaitingForInput` phase. The dashboard displays the question and provides an input field. Submit your response via the UI or through the `SendHumanInput` API endpoint.
