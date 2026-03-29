## ADDED Requirements

### Requirement: Agents can call semantic_search as a sidecar RPC tool
The `AgentSidecarService` proto SHALL include a `SemanticSearch` RPC method. When called, the sidecar gateway SHALL forward the query to the cudgel service and return ranked code symbol results. The tool SHALL be callable by agents at any point during run execution.

#### Scenario: Agent calls semantic_search and receives code symbols
- **WHEN** an agent sends a `SemanticSearch` RPC with query "how does authentication work" and limit 5
- **THEN** the sidecar forwards the query to the cudgel `/search` endpoint
- **AND** the response contains up to 5 `CodeChunk` messages
- **AND** each `CodeChunk` includes `name`, `kind`, `file`, `line`, `snippet`, and `score`

#### Scenario: Agent calls semantic_search when cudgel is unavailable
- **WHEN** an agent sends a `SemanticSearch` RPC
- **AND** the cudgel endpoint is unreachable or returns a non-200 status
- **THEN** the sidecar returns an empty `SemanticSearchResponse` (no error returned to agent)
- **AND** the failure is logged at WARN level

#### Scenario: semantic_search with empty query returns InvalidArgument error
- **WHEN** an agent sends a `SemanticSearch` RPC with an empty query string
- **THEN** the sidecar returns a gRPC `InvalidArgument` error

### Requirement: semantic_search tool is disabled when CUDGEL_ENDPOINT is not set
When the `CUDGEL_ENDPOINT` environment variable is absent or empty on the sidecar container, the `SemanticSearch` RPC SHALL return an empty response immediately without attempting any HTTP calls.

#### Scenario: Tool silently no-ops without endpoint config
- **WHEN** the sidecar container has no `CUDGEL_ENDPOINT` env var
- **AND** an agent calls `SemanticSearch`
- **THEN** the response is an empty `SemanticSearchResponse` with no error

### Requirement: semantic_search result limit is bounded
The `SemanticSearchRequest` SHALL accept a `limit` field (int32). If `limit` is 0 or unset, the default SHALL be 10. If `limit` exceeds 50, it SHALL be clamped to 50.

#### Scenario: Default limit is applied when unset
- **WHEN** an agent calls `SemanticSearch` with `limit = 0`
- **THEN** up to 10 results are returned

#### Scenario: Limit is clamped to maximum
- **WHEN** an agent calls `SemanticSearch` with `limit = 100`
- **THEN** at most 50 results are returned
