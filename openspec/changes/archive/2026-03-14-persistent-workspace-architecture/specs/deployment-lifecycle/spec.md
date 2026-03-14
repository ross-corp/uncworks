## ADDED Requirements

### Requirement: Deployment-based agent run compute
Each agent run SHALL use a Kubernetes Deployment (replicas 0 or 1) instead of a bare Pod for compute lifecycle management.

#### Scenario: Run creates Deployment and PVC
- **WHEN** an agent run is created
- **THEN** a Deployment with replicas=1 and a PVC (2Gi, local-path) are created
- **AND** the PVC is mounted at `/workspace` in all containers

#### Scenario: Completion scales Deployment to zero
- **WHEN** the agent run reaches a terminal phase (Succeeded, Failed, Cancelled)
- **THEN** the Deployment is scaled to replicas=0
- **AND** the PVC remains intact with all workspace data

#### Scenario: Deployment persists as compute identity
- **WHEN** a run is completed and scaled to 0
- **THEN** the Deployment object still exists in the cluster
- **AND** it can be scaled back to 1 for debug access

### Requirement: local-path-provisioner storage
The k0s cluster SHALL have local-path-provisioner installed to provide PVC support for agent workspaces.

#### Scenario: PVC provisioned on creation
- **WHEN** a PVC with storageClass `local-path` is created
- **THEN** a PersistentVolume is automatically provisioned on the host at `/opt/local-path-provisioner/`

#### Scenario: PVC data survives Pod deletion
- **WHEN** a Pod mounting a PVC is deleted (Deployment scaled to 0)
- **THEN** the PVC data remains on disk at the provisioned host path
