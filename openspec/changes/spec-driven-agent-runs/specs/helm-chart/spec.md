## MODIFIED Requirements

### Requirement: Worker deployment configures pipeline settings
The worker Helm template SHALL pass pipeline configuration (max retries, planning timeout, verification model) to the Temporal worker via environment variables.

#### Scenario: Pipeline config passed to worker
- **WHEN** the Helm chart is deployed with `pipeline.maxRetries=3` and `pipeline.planTimeout=120`
- **THEN** the worker container has `AOT_PIPELINE_MAX_RETRIES=3` and `AOT_PIPELINE_PLAN_TIMEOUT=120` environment variables set
