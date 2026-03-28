## ADDED Requirements

### Requirement: Local Helm values preset
The repository SHALL include a `deploy/helm/values.local.yaml` file providing defaults optimized for local development on a laptop cluster.

#### Scenario: File exists in repo
- **WHEN** the repository is cloned
- **THEN** `deploy/helm/values.local.yaml` exists and is valid YAML

### Requirement: NodePort service exposure in local values
The local values preset SHALL configure the web dashboard and API server as NodePort services on known, stable port numbers.

#### Scenario: Web UI accessible via NodePort
- **WHEN** the chart is installed with `values.local.yaml` on Docker Desktop or OrbStack
- **THEN** the web UI is accessible at `http://localhost:30300`

#### Scenario: API accessible via NodePort
- **WHEN** the chart is installed with `values.local.yaml`
- **THEN** the gRPC API is accessible via port-forward to the configured ClusterIP service (NodePort not required for API)

### Requirement: Ollama disabled by default in local values
The local values preset SHALL set Ollama to disabled, with a comment indicating how to enable it.

#### Scenario: Ollama not deployed by default
- **WHEN** the chart is installed with `values.local.yaml` and no override
- **THEN** no Ollama deployment or service is created in the cluster

### Requirement: Reduced resource requests in local values
The local values preset SHALL specify lower CPU and memory requests than the base values, appropriate for a development laptop.

#### Scenario: Pods schedule on constrained cluster
- **WHEN** the chart is installed with `values.local.yaml` on a cluster with 4 CPU / 4Gi memory
- **THEN** all pods reach Running state without Pending due to resource pressure

### Requirement: Storage class left as default in local values
The local values preset SHALL NOT hardcode a storage class, allowing the cluster's default storage class to be used.

#### Scenario: PVC uses cluster default
- **WHEN** the chart is installed with `values.local.yaml` on any of Docker Desktop, OrbStack, Rancher Desktop, Colima, k3d, or kind
- **THEN** PVCs are provisioned using the cluster's default storage class without manual configuration
