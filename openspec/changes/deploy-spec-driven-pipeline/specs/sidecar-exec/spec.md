## Purpose

Define a lightweight command execution RPC on the sidecar for running bash commands in the agent workspace without spawning a full AI agent.

## ADDED Requirements

### Requirement: Sidecar provides ExecCommand RPC
The sidecar SHALL expose an `ExecCommand` RPC that runs a bash command in the workspace and returns stdout, stderr, and exit code.

#### Scenario: Successful command execution
- **WHEN** `ExecCommand` is called with `command: "ls -la /workspace"`
- **THEN** the response contains stdout with the directory listing, empty stderr, and exit code 0

#### Scenario: Failed command execution
- **WHEN** `ExecCommand` is called with a command that fails (e.g., `command: "false"`)
- **THEN** the response contains exit code 1 and any stderr output

#### Scenario: Command timeout
- **WHEN** `ExecCommand` is called with `timeout_seconds: 5` and the command runs longer than 5 seconds
- **THEN** the command is killed and the response contains a non-zero exit code

### Requirement: ExecCommand runs in the workspace directory
The command SHALL execute with the working directory set to the workspace root (`/workspace`) by default, or to the `working_dir` field if specified.

#### Scenario: Default working directory
- **WHEN** `ExecCommand` is called without `working_dir`
- **THEN** the command runs in `/workspace`

#### Scenario: Custom working directory
- **WHEN** `ExecCommand` is called with `working_dir: "/workspace/src/my-repo"`
- **THEN** the command runs in that directory

### Requirement: Verification activities use ExecCommand instead of agent spawning
The `VerifyRun` activity SHALL use `ExecCommand` for running `openspec` CLI commands and test suites instead of spawning full AI agents.

#### Scenario: OpenSpec validation via ExecCommand
- **WHEN** the verification gate runs `openspec validate --json`
- **THEN** it uses `ExecCommand` RPC, not `StartAgent`
- **AND** the response is parsed as JSON to extract validation results
