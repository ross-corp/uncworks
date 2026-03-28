## ADDED Requirements

### Requirement: uncworks binary is the single user-facing entrypoint
The system SHALL provide a single `uncworks` binary at `cmd/uncworks/` that exposes all user-facing operations as subcommands. The binary SHALL be buildable for darwin/arm64, darwin/amd64, linux/amd64, and linux/arm64 with CGO_ENABLED=0.

#### Scenario: Build for all platforms
- **WHEN** `GOOS=darwin GOARCH=arm64 go build ./cmd/uncworks` is run
- **THEN** a statically-linked `uncworks` binary is produced with no CGO dependencies

### Requirement: uncworks setup subcommand
The system SHALL provide an `uncworks setup` subcommand that runs the interactive setup wizard to deploy UNCWORKS into a local Kubernetes cluster.

#### Scenario: Setup with active context
- **WHEN** `uncworks setup` is run and a valid kubeconfig context is active
- **THEN** the wizard validates the context, prompts for required configuration, deploys the Helm chart, and prints the web UI URL

#### Scenario: Setup with no cluster available
- **WHEN** `uncworks setup` is run and no kubeconfig contexts exist
- **THEN** the CLI prints instructions for installing a local Kubernetes cluster and exits with a non-zero code

### Requirement: uncworks teardown subcommand
The system SHALL provide an `uncworks teardown` subcommand that uninstalls the Helm release from the active cluster.

#### Scenario: Teardown removes release
- **WHEN** `uncworks teardown` is run
- **THEN** the Helm release is uninstalled; PVCs are NOT deleted by default (data preserved)

#### Scenario: Teardown with --purge flag
- **WHEN** `uncworks teardown --purge` is run
- **THEN** the Helm release is uninstalled AND all PVCs in the UNCWORKS namespace are deleted

### Requirement: uncworks status subcommand
The system SHALL provide an `uncworks status` subcommand that shows the health of the deployed UNCWORKS stack.

#### Scenario: All pods healthy
- **WHEN** `uncworks status` is run and all UNCWORKS pods are running
- **THEN** each component (apiserver, worker, controller, web, bff) is listed with its status and the web UI URL is shown

#### Scenario: Pod not ready
- **WHEN** `uncworks status` is run and a pod is in Pending or CrashLoopBackOff
- **THEN** the affected component is flagged with its pod status and last event

### Requirement: uncworks open subcommand
The system SHALL provide an `uncworks open` subcommand that starts a `kubectl port-forward` subprocess and opens the web UI in the default browser.

#### Scenario: Open starts port-forward and browser
- **WHEN** `uncworks open` is run
- **THEN** a `kubectl port-forward` subprocess is started for the web service, and the default browser is opened to the forwarded URL

#### Scenario: Open cleans up stale port-forward
- **WHEN** `uncworks open` is run and a previous port-forward PID file exists at `~/.config/uncworks/port-forward.pid`
- **THEN** the stale process is killed before starting a new one

### Requirement: uncworks connect subcommand
The system SHALL provide an `uncworks connect <address>` subcommand that stores a remote gRPC server address for use by `uncworks tui`.

#### Scenario: Connect stores address
- **WHEN** `uncworks connect grpc.example.com:50055` is run
- **THEN** the address is written to `~/.config/uncworks/config.yaml` under `server.address`

### Requirement: Config stored in XDG directory
The CLI SHALL store all configuration in `$XDG_CONFIG_HOME/uncworks/` (defaulting to `~/.config/uncworks/`) on both macOS and Linux.

#### Scenario: Config written to XDG path
- **WHEN** any `uncworks` command writes configuration
- **THEN** the file is written under `~/.config/uncworks/` when `$XDG_CONFIG_HOME` is unset

### Requirement: Prerequisites validation
The CLI SHALL validate that `kubectl` and `helm` are available in PATH before running any cluster operations, and print clear install instructions if missing.

#### Scenario: Missing kubectl
- **WHEN** `uncworks setup` is run and `kubectl` is not in PATH
- **THEN** the CLI prints "kubectl not found. Install from https://kubernetes.io/docs/tasks/tools/" and exits with a non-zero code
