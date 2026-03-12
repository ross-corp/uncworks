# Agent Run Lifecycle

This document covers the complete lifecycle of an agent run — from creation through execution to cleanup — including phase transitions, HITL interactions, multi-agent workflows, and failure modes.

## Phase State Machine

```mermaid
stateDiagram-v2
    [*] --> Pending: CreateAgentRun
    Pending --> Running: Pod provisioned + hydrated
    Pending --> Failed: Pod creation failed
    Running --> WaitingForInput: Agent calls ask_human
    WaitingForInput --> Running: SendHumanInput received
    Running --> Succeeded: Agent completed task
    Running --> Failed: Error or TTL exceeded
    Running --> Cancelled: CancelAgentRun
    WaitingForInput --> Cancelled: CancelAgentRun
    Pending --> Cancelled: CancelAgentRun
    Succeeded --> [*]
    Failed --> [*]
    Cancelled --> [*]
```

| Phase | Meaning | Terminal |
|-------|---------|---------|
| **Pending** | Run created, pod not yet provisioned or still hydrating | No |
| **Running** | Agent is actively executing | No |
| **WaitingForInput** | Agent paused, awaiting human response via HITL | No |
| **Succeeded** | Agent completed the task | Yes |
| **Failed** | Agent errored, init failed, or TTL expired | Yes |
| **Cancelled** | Cancelled by user | Yes |

## Creation Flow

```mermaid
sequenceDiagram
    participant Client as Client
    participant API as API Server
    participant Ctrl as Controller
    participant Temp as Temporal
    participant Worker as Temporal Worker
    participant K8s as Kubernetes

    Client->>API: CreateAgentRun(spec)
    API->>API: Validate spec (protovalidate)
    API->>API: Store AgentRun in memory
    API-->>Client: AgentRun (phase: Pending)

    Note over Ctrl: Reconcile loop detects new CRD

    Ctrl->>Temp: ExecuteWorkflow(AgentRunWorkflow, spec)
    Temp-->>Ctrl: workflowID
    Ctrl->>K8s: Annotate CRD with workflow-id
    Ctrl->>K8s: Add finalizer

    Note over Worker: Workflow begins execution
```

## Workflow Execution

The Temporal workflow (`AgentRunWorkflow`) orchestrates the full lifecycle through a sequence of activities:

```mermaid
sequenceDiagram
    participant WF as Workflow
    participant LL as LiteLLM
    participant K8s as Kubernetes
    participant SC as Sidecar

    rect rgb(40, 40, 50)
        Note over WF: Step 1: Provision LLM Access
        WF->>LL: ProvisionLLMKey(tier, budget)
        LL-->>WF: virtualKey
    end

    rect rgb(40, 40, 50)
        Note over WF: Step 2: Create Pod
        WF->>K8s: CreateAgentPod(spec, key)
        K8s-->>WF: podName
    end

    rect rgb(40, 40, 50)
        Note over WF: Step 3: Wait for Hydration
        loop Poll init container (timeout: 10m)
            WF->>K8s: Get pod status
            K8s-->>WF: init container state
        end
    end

    rect rgb(40, 40, 50)
        Note over WF: Step 4: Start Agent
        WF->>SC: StartAgent(prompt)
        SC-->>WF: OK
    end

    rect rgb(40, 40, 50)
        Note over WF: Step 5: Monitor + Handle Signals
        loop Poll every 5s until terminal
            WF->>SC: GetStatus()
            SC-->>WF: phase, message
        end
    end

    rect rgb(50, 30, 30)
        Note over WF: Cleanup (always runs via defer)
        WF->>LL: RevokeLLMKey
        WF->>K8s: CleanupPod (delete)
    end
```

### Activity Timeouts

| Activity | Timeout | Retries |
|----------|---------|---------|
| ProvisionLLMKey | 5 min | 3 (exponential backoff, max 30s) |
| CreateAgentPod | 5 min | 3 |
| WaitForHydration | 10 min | 3 |
| StartAgent | 5 min | 3 |
| GetAgentStatus | 5 min | 3 |
| CleanupPod | 30 sec | 3 |
| RevokeLLMKey | 30 sec | 3 |

## Hydration

The init container provisions the workspace before the agent starts:

```mermaid
flowchart TD
    Start([Init Container Starts]) --> Clone

    Clone[Clone repo as bare repository<br/>/workspace/.bare]
    Clone --> Worktree

    Worktree[Create git worktree<br/>Branch: aot/main<br/>Path: /workspace/src]
    Worktree --> CheckDevbox

    CheckDevbox{devboxConfig<br/>specified?}
    CheckDevbox -->|Yes| Devbox[Run devbox install<br/>in /workspace/src]
    CheckDevbox -->|No| Done

    Devbox --> Done([Exit 0 — Main containers start])
```

The bare repo pattern allows multiple agents to work on the same repository concurrently with isolated worktrees.

## Human-in-the-Loop (HITL)

When an agent needs clarification, it calls `ask_human` which triggers a signal-based pause/resume cycle:

```mermaid
sequenceDiagram
    participant Agent
    participant Ext as pi-aot-extension
    participant SC as Sidecar
    participant WF as Workflow
    participant API as API Server
    participant Client

    Agent->>Ext: ask_human("Which database?")
    Ext->>SC: Process state → WaitingForInput
    SC->>WF: GetStatus() → WaitingForInput

    Note over WF: Phase transitions to WaitingForInput
    Note over WF: Controller syncs to CRD

    API-->>Client: WatchAgentRun event: WaitingForInput

    Client->>API: SendHumanInput("Use PostgreSQL")
    API->>WF: SignalWorkflow(SignalHumanInput, input)

    WF->>SC: ForwardHumanInput("Use PostgreSQL")
    SC->>Agent: Write to stdin

    Note over Agent: Resumes execution
    SC->>WF: GetStatus() → Running
```

HITL is delivered via Temporal signals, which means:
- No polling or direct routing required
- Signals are durable — they survive worker restarts
- The workflow blocks on a selector until the signal arrives or TTL expires

## TTL Enforcement

Each agent run has a TTL (default: 3600 seconds). The workflow uses a Temporal timer that races against the status polling loop:

```mermaid
flowchart TD
    Start([Workflow monitoring phase]) --> Select

    Select{Temporal Selector}
    Select -->|Status poll fires| CheckPhase{Agent phase?}
    Select -->|TTL timer fires| TTLExpired[StopAgent → Failed<br/>'Exceeded TTL']
    Select -->|Cancel signal| CancelAgent[StopAgent → Cancelled]
    Select -->|HumanInput signal| Forward[ForwardHumanInput<br/>→ Running]

    CheckPhase -->|Running / WaitingForInput| Select
    CheckPhase -->|Completed| Succeeded([Succeeded])
    CheckPhase -->|Failed| Failed([Failed])

    TTLExpired --> Cleanup([Cleanup: revoke key + delete pod])
    CancelAgent --> Cleanup
    Succeeded --> Cleanup
    Failed --> Cleanup
    Forward --> Select
```

## Multi-Agent Workflows

A senior agent can spawn junior agents via the `spawn_junior` tool exposed by the pi-aot-extension:

```mermaid
sequenceDiagram
    participant Senior as Senior Agent
    participant Ext as pi-aot-extension
    participant API as Control Plane
    participant Junior as Junior Agent Pod

    Senior->>Ext: spawn_junior("Write auth tests")
    Ext->>API: CreateAgentRun(child spec)
    API-->>Ext: junior run ID

    Note over API: Child inherits parent config:<br/>backend, repo, branch, image, TTL

    API->>Junior: Provision + start (normal lifecycle)

    par Junior executes independently
        Junior->>Junior: Clone, hydrate, run agent
    and Senior continues
        Senior->>Senior: Continue refactoring
    end

    Senior->>API: GetAgentRun(junior ID)
    API-->>Senior: status: Succeeded
```

Child runs are labeled for tracking:
- `aot.uncworks.io/parent: <parent-name>`
- `aot.uncworks.io/role: junior`
- `aot.uncworks.io/managed: true`

## Cancellation

Cancellation is cooperative — the workflow sends SIGINT first, allowing the agent to exit gracefully:

```mermaid
sequenceDiagram
    participant Client
    participant API as API Server
    participant Temp as Temporal
    participant WF as Workflow
    participant SC as Sidecar
    participant Agent

    Client->>API: CancelAgentRun(id)
    API->>Temp: CancelWorkflow(workflowID)
    API->>API: Update local state → Cancelled

    Temp->>WF: Cancel signal
    WF->>SC: StopAgent(force=false)
    SC->>Agent: SIGINT
    Agent->>Agent: Graceful shutdown

    Note over WF: Phase → Cancelled
    Note over WF: Defer cleanup runs
    WF->>SC: RevokeLLMKey
    WF->>SC: CleanupPod (delete)
```

## Failure Modes

| Failure | Detection | Recovery |
|---------|-----------|----------|
| Pod creation fails | `CreateAgentPod` activity errors | Retried 3x with backoff. Workflow transitions to Failed. |
| Init container fails | `WaitForHydration` sees non-zero exit | Workflow transitions to Failed. Pod cleaned up. |
| Agent process crashes | `GetAgentStatus` returns Failed | Workflow transitions to Failed. Pod cleaned up. |
| TTL expires | Temporal timer fires | `StopAgent` called, workflow transitions to Failed. |
| Sidecar unreachable | Activity timeout (5 min) | Retried 3x. On exhaustion, workflow fails. |
| Worker restarts | Temporal durable execution | Workflow resumes from last checkpoint. No state lost. |
| Controller restarts | Kubernetes reconcile loop | Re-syncs all non-terminal CRDs on next reconcile. |
| LiteLLM key provision fails | Activity error | Retried 3x. On exhaustion, workflow fails before pod creation. |

## Controller Reconcile Loop

The controller bridges Kubernetes CRDs and Temporal workflows on a 30-second reconcile interval:

```mermaid
flowchart TD
    Reconcile([Reconcile triggered]) --> CheckDeletion

    CheckDeletion{CRD being deleted?}
    CheckDeletion -->|Yes| Cancel[CancelWorkflow]
    Cancel --> RemoveFinalizer[Remove finalizer]
    RemoveFinalizer --> Done([Done])

    CheckDeletion -->|No| CheckAnnotation{Has workflow-id<br/>annotation?}
    CheckAnnotation -->|No| StartWorkflow[ExecuteWorkflow]
    StartWorkflow --> Annotate[Annotate CRD with workflow-id]
    Annotate --> Requeue([Requeue in 30s])

    CheckAnnotation -->|Yes| QueryState[QueryWorkflow: get-state]
    QueryState --> MapPhase[Map workflow phase → CRD phase]
    MapPhase --> CheckChanged{Phase changed?}
    CheckChanged -->|Yes| UpdateCRD[Update CRD status]
    CheckChanged -->|No| CheckTerminal

    UpdateCRD --> CheckTerminal{Terminal phase?}
    CheckTerminal -->|Yes| Done
    CheckTerminal -->|No| Requeue
```

### Phase Mapping

| Workflow Phase | CRD Phase |
|---------------|-----------|
| Pending, Creating, Hydrating | Pending |
| Running | Running |
| WaitingForInput | WaitingForInput |
| Succeeded | Succeeded |
| Failed | Failed |
| Cancelling, Cancelled | Cancelled |
