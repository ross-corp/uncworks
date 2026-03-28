## ADDED Requirements

### Requirement: Kubeconfig context detection
The setup wizard SHALL enumerate all contexts in `~/.kube/config`, identify the active context, and allow the user to select a different one before proceeding.

#### Scenario: Single context auto-selected
- **WHEN** `uncworks setup` is run and exactly one kubeconfig context exists
- **THEN** the wizard uses that context without prompting, displaying its name for confirmation

#### Scenario: Multiple contexts — user selects
- **WHEN** `uncworks setup` is run and multiple kubeconfig contexts exist
- **THEN** an interactive list is displayed with the active context highlighted; the user selects one before proceeding

### Requirement: Context identity display
The wizard SHALL display the cluster server URL alongside each context name so the user can verify they are targeting the correct cluster.

#### Scenario: Context shown with server URL
- **WHEN** the context selection list is rendered
- **THEN** each entry shows `<context-name>  (<server-url>)`

### Requirement: Cluster resource preflight check
The wizard SHALL check that the selected cluster has sufficient allocatable CPU and memory before installing.

#### Scenario: Sufficient resources
- **WHEN** the cluster has ≥4 allocatable CPU and ≥4Gi allocatable memory
- **THEN** the preflight passes and installation proceeds

#### Scenario: Marginal resources — warning
- **WHEN** the cluster has ≥2 allocatable CPU and ≥2Gi memory but below the recommended threshold
- **THEN** the wizard displays a warning ("Resources below recommended minimum — install may be slow") and asks for confirmation

#### Scenario: Insufficient resources — hard fail
- **WHEN** the cluster has <2 allocatable CPU or <2Gi allocatable memory
- **THEN** the wizard exits with an error explaining the minimum requirements

### Requirement: Required configuration collection
The wizard SHALL interactively collect required UNCWORKS configuration values that are not already present in `~/.config/uncworks/config.yaml`.

#### Scenario: LLM API key prompt
- **WHEN** no LLM API key is stored and `--llm-key` flag is not provided
- **THEN** the wizard prompts for an OpenRouter or OpenAI API key with masked input

#### Scenario: GitHub token prompt
- **WHEN** no GitHub token is stored and `--github-token` flag is not provided
- **THEN** the wizard prompts for a GitHub personal access token with masked input

#### Scenario: Temporal host prompt
- **WHEN** no Temporal host is stored and `--temporal-host` flag is not provided
- **THEN** the wizard prompts for the Temporal gRPC address (e.g., `temporal:7233`)

### Requirement: Non-interactive mode via flags
All wizard prompts SHALL be bypassable via CLI flags for scripted use.

#### Scenario: Fully scripted setup
- **WHEN** `uncworks setup --context docker-desktop --llm-key sk-xxx --github-token ghp_xxx --temporal-host temporal:7233` is run
- **THEN** the wizard skips all interactive prompts and proceeds directly to installation

### Requirement: Helm install with idempotency
The wizard SHALL use `helm upgrade --install` so that re-running `uncworks setup` upgrades an existing release rather than failing.

#### Scenario: First install
- **WHEN** `uncworks setup` is run and no Helm release named `uncworks` exists
- **THEN** `helm install uncworks oci://ghcr.io/uncworks/charts/aot` is executed with collected values

#### Scenario: Re-run upgrades
- **WHEN** `uncworks setup` is run and a Helm release named `uncworks` already exists
- **THEN** `helm upgrade uncworks oci://ghcr.io/uncworks/charts/aot` is executed, updating the release

### Requirement: Post-install URL output
After successful installation the wizard SHALL print the web UI URL and the gRPC API address.

#### Scenario: Install completes
- **WHEN** the Helm release is successfully installed
- **THEN** the wizard prints the web UI URL and instructions for running `uncworks open`

### Requirement: No cluster path — helpful exit
When no Kubernetes contexts are found, the wizard SHALL print platform-appropriate install recommendations and exit cleanly.

#### Scenario: No clusters on macOS
- **WHEN** `uncworks setup` is run on macOS with no kubeconfig contexts
- **THEN** the CLI prints recommendations for Docker Desktop, OrbStack, and Rancher Desktop and exits

#### Scenario: No clusters on Linux
- **WHEN** `uncworks setup` is run on Linux with no kubeconfig contexts
- **THEN** the CLI prints recommendations for k3d and kind and exits
