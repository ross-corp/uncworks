# github-auth-provider Specification

## Purpose
TBD - created by archiving change github-token-provider. Update Purpose after archive.
## Requirements
### Requirement: Token provider interface replaces direct env var access
The system SHALL use a GitHubTokenProvider interface for all GitHub API authentication, replacing direct os.Getenv("GITHUB_TOKEN") calls.

#### Scenario: PATProvider returns configured token
- **WHEN** a PATProvider is created with token "ghp_abc123"
- **THEN** calling Token() SHALL return "ghp_abc123"

#### Scenario: PATProvider returns error when empty
- **WHEN** a PATProvider is created with an empty string
- **THEN** calling Token() SHALL return an error

#### Scenario: All GitHub consumers use provider
- **WHEN** the codebase is searched for os.Getenv("GITHUB_TOKEN")
- **THEN** zero direct calls SHALL exist outside the provider package

### Requirement: Agent pods SHALL NOT have GitHub credentials
The system SHALL NOT inject GITHUB_TOKEN into the agent or sidecar containers. Only the init container (hydration) SHALL receive it for clone operations.

#### Scenario: Init container has token for private repo clone
- **WHEN** an agent pod is created with a configured github.tokenSecretName
- **THEN** the init container SHALL have GITHUB_TOKEN set from the k8s Secret
- **AND** the agent container SHALL NOT have GITHUB_TOKEN in its environment
- **AND** the sidecar container SHALL NOT have GITHUB_TOKEN in its environment

### Requirement: Git push runs worker-side via PVC
The system SHALL perform git commit and push operations from the Temporal worker using the PVC host path, not from inside the agent pod.

#### Scenario: PushChanges uses worker-side git
- **WHEN** the PushChanges activity executes after successful verification
- **THEN** it SHALL read the workspace from the PVC host path
- **AND** it SHALL configure git credentials using the token provider
- **AND** it SHALL NOT call execInSidecar for git push

### Requirement: GitHub token wired via Helm Secret
The system SHALL support configuring the GitHub token as a k8s Secret reference in the Helm chart.

#### Scenario: Secret configured
- **WHEN** github.tokenSecretName is set in Helm values
- **THEN** the worker and apiserver deployments SHALL mount the token from that Secret

