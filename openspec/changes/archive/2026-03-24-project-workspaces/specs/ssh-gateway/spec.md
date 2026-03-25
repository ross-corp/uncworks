## ADDED Requirements

### Requirement: SSH Gateway Deployment
The system SHALL deploy a single SSH gateway pod exposed via a NodePort service on port 30022. The gateway SHALL accept incoming SSH connections and route them based on the connecting username.

#### Scenario: Gateway is reachable on NodePort
- **WHEN** the SSH gateway pod is deployed
- **THEN** it SHALL accept SSH connections on NodePort 30022 from outside the cluster

### Requirement: Username-Based Routing
The SSH gateway SHALL interpret the SSH username as the target project name. The gateway SHALL look up the Project resource matching the username to determine the target IDE pod.

#### Scenario: Connection routed to correct project IDE pod
- **WHEN** a user connects with `ssh my-project@gateway-host -p 30022`
- **THEN** the gateway SHALL resolve `my-project` to the corresponding Project resource and route the session to that project's IDE pod

#### Scenario: Unknown project name rejected
- **WHEN** a user connects with a username that does not match any Project resource
- **THEN** the gateway SHALL reject the connection with an authentication error

### Requirement: SSH Key Verification
The gateway SHALL verify the connecting user's SSH public key against the `authorizedKeys` list in the target Project CRD. Connections with keys not present in the project's `authorizedKeys` SHALL be rejected.

#### Scenario: Valid key accepted
- **WHEN** a user connects with an SSH key that matches an entry in the target project's `authorizedKeys`
- **THEN** the gateway SHALL authenticate the session and proceed with routing

#### Scenario: Invalid key rejected
- **WHEN** a user connects with an SSH key not listed in the target project's `authorizedKeys`
- **THEN** the gateway SHALL reject the connection with an authentication failure

### Requirement: IDE Pod Wake and Proxy
If the target IDE pod is scaled to zero, the gateway SHALL trigger a scale-up and wait for the pod to become ready before proxying. The gateway SHALL proxy the authenticated SSH session to the IDE pod's sshd on port 2222.

#### Scenario: Gateway wakes idle IDE pod before proxying
- **WHEN** an authenticated user connects to a project whose IDE pod is scaled to zero
- **THEN** the gateway SHALL scale the IDE pod to one replica, wait for it to become ready, and then proxy the SSH session to port 2222 on the IDE pod
