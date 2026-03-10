## ADDED Requirements

### Requirement: Ollama Helm Deployment
Ollama SHALL be deployable to k0s via Helm chart as a LiteLLM backend.

#### Scenario: Deploying Ollama to k0s
- **WHEN** an operator deploys Ollama using the documented Helm chart
- **THEN** Ollama SHALL run as a Kubernetes deployment accessible within the cluster

### Requirement: Ollama Deployment Documentation
Ollama deployment documentation SHALL be provided in `deploy/ollama/`.

#### Scenario: Operator follows deployment guide
- **WHEN** an operator reads the deployment documentation in `deploy/ollama/`
- **THEN** they SHALL have sufficient information to deploy Ollama to k0s with appropriate resource configuration

### Requirement: In-Cluster DNS Accessibility
Ollama SHALL serve models accessible to LiteLLM via in-cluster DNS at `http://ollama:11434`.

#### Scenario: LiteLLM connects to Ollama
- **WHEN** LiteLLM routes a request to an Ollama-backed model
- **THEN** LiteLLM SHALL reach Ollama via `http://ollama:11434` (or the configured Ollama service DNS)

### Requirement: CI/Testing Model
For CI/testing, Ollama SHALL serve a minimal model (e.g., `qwen2.5:0.5b`) for fast inference.

#### Scenario: Running integration tests
- **WHEN** Ollama is deployed in a CI or testing environment
- **THEN** a minimal model such as `qwen2.5:0.5b` SHALL be available for fast, low-resource inference
- **AND** this SHALL enable full agent lifecycle tests without paid API calls

### Requirement: Development Model
For development, Ollama SHALL serve a capable model (e.g., `llama3.1:8b`).

#### Scenario: Local development workflow
- **WHEN** Ollama is deployed in a development environment
- **THEN** a capable model such as `llama3.1:8b` SHALL be available for realistic agent testing

### Requirement: Model Pull as Post-Deployment Step
Ollama model pull SHALL be documented as a post-deployment step.

#### Scenario: Pulling models after deployment
- **WHEN** Ollama is deployed via Helm
- **THEN** the documentation SHALL describe how to pull the required model(s) using `ollama pull` as a post-deployment step
- **AND** model pulls SHALL NOT be automated in the Helm chart to avoid blocking deployment on large downloads

### Requirement: LiteLLM Ollama Model Prefix
LiteLLM configuration SHALL use `ollama_chat/` prefix for Ollama models.

#### Scenario: Configuring Ollama models in LiteLLM
- **WHEN** Ollama models are added to LiteLLM's `model_list`
- **THEN** the model names SHALL use the `ollama_chat/` prefix (e.g., `ollama_chat/llama3.1:8b`)
- **AND** the `api_base` SHALL point to the Ollama in-cluster service URL
