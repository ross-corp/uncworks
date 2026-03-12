## ADDED Requirements

### Requirement: devbox.json declares all development dependencies
The `devbox.json` SHALL include every tool required for local development, testing, building, linting, and infrastructure management. A developer SHALL NOT need to install any tool manually after running `devbox shell`.

#### Scenario: Fresh clone and devbox shell
- **WHEN** a developer clones the repo and runs `devbox shell`
- **THEN** all of the following are available on PATH: go, node, npm, npx, task, protoc, protoc-gen-go, protoc-gen-go-grpc, kubectl, k0sctl, psql, lefthook, golangci-lint, buf, grpcurl, helm, temporal, setup-envtest

### Requirement: Devbox includes Go development tools
The following Go ecosystem tools SHALL be declared in devbox.json: `go`, `golangci-lint`, `setup-envtest`.

#### Scenario: Go lint available
- **WHEN** a developer runs `golangci-lint run` in devbox shell
- **THEN** the command executes without "command not found"

#### Scenario: envtest assets resolvable
- **WHEN** a developer runs `setup-envtest use --print path`
- **THEN** the command returns a valid path to envtest binaries

### Requirement: Devbox includes protobuf tools
The following protobuf tools SHALL be declared in devbox.json: `protobuf` (protoc), `go-protobuf` (protoc-gen-go), `protoc-gen-go-grpc`, `buf`.

#### Scenario: buf available
- **WHEN** a developer runs `buf --version` in devbox shell
- **THEN** the command executes successfully

### Requirement: Devbox includes infrastructure tools
The following infrastructure tools SHALL be declared in devbox.json: `kubectl`, `k0sctl`, `kubernetes-helm`, `temporal-cli`, `grpcurl`, `postgresql`.

#### Scenario: Helm available
- **WHEN** a developer runs `helm version` in devbox shell
- **THEN** the command executes successfully

#### Scenario: Temporal CLI available
- **WHEN** a developer runs `temporal --version` in devbox shell
- **THEN** the command executes successfully

### Requirement: Devbox includes task runner
The `go-task` package SHALL be declared in devbox.json so the `task` binary is available without global installation.

#### Scenario: Task runner available
- **WHEN** a developer runs `task --list` in devbox shell
- **THEN** the command lists all available tasks

### Requirement: Devbox init_hook installs git hooks
The devbox.json `init_hook` SHALL run `lefthook install` automatically so hooks are installed on every shell entry.

#### Scenario: Hooks installed on shell entry
- **WHEN** a developer runs `devbox shell`
- **THEN** lefthook hooks are installed (or confirmed already installed)
