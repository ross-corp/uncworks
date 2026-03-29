## ADDED Requirements

### Requirement: Hydration seeds codebase symbol context from cudgel at run start
After workspace provisioning and before agent startup, the hydrator SHALL call the cudgel `/search` endpoint using the run's prompt as the query. The top-K results SHALL be written to `.aot/context/codebase.md` in the agent workspace. If cudgel is unreachable or returns no results, the file SHALL NOT be written and the run SHALL proceed normally.

#### Scenario: Cudgel returns results for the prompt
- **WHEN** a new agent run starts with prompt "fix the authentication middleware"
- **AND** `CUDGEL_ENDPOINT` is set and cudgel is reachable
- **THEN** the hydrator calls `POST /search` with the prompt text and limit K
- **AND** `.aot/context/codebase.md` is written with formatted symbol summaries
- **AND** the agent starts normally after the file is written

#### Scenario: Cudgel is unreachable at hydration time
- **WHEN** a new agent run starts
- **AND** the cudgel endpoint is unreachable (connection refused, timeout)
- **THEN** no `.aot/context/codebase.md` is written
- **AND** a warning is logged
- **AND** the run proceeds to agent startup without codebase context

#### Scenario: Cudgel returns no matching symbols
- **WHEN** a new agent run starts with a prompt that has no relevant symbols
- **THEN** no `.aot/context/codebase.md` is written
- **AND** the run proceeds normally

#### Scenario: Codebase context hydration does not block run on timeout
- **WHEN** the cudgel HTTP call exceeds 5 seconds
- **THEN** the call is cancelled
- **AND** no codebase context file is written
- **AND** the run proceeds to agent startup without delay beyond the timeout

### Requirement: Senior agents receive more codebase symbols at hydration
When the run's agent type is "senior" or "orchestrator", the codebase context hydration query SHALL use K=20. For all other agent types, K SHALL be 10.

#### Scenario: Senior agent receives expanded codebase context
- **WHEN** a run with agent type "senior" starts
- **AND** cudgel returns results
- **THEN** `.aot/context/codebase.md` contains up to 20 symbol entries

#### Scenario: Regular agent receives standard codebase context
- **WHEN** a run with agent type "worker" starts
- **AND** cudgel returns results
- **THEN** `.aot/context/codebase.md` contains up to 10 symbol entries

### Requirement: Codebase context file is formatted for agent consumption
The `.aot/context/codebase.md` file SHALL be markdown formatted. Each symbol entry SHALL include: symbol name, kind (function/struct/etc), file path, line number, and a brief snippet. The file SHALL include a header explaining it is a semantic search result seeded from cudgel. Total file size SHALL NOT exceed 4,000 tokens.

#### Scenario: Codebase context file structure
- **WHEN** codebase context is written with 5 symbol results
- **THEN** the file begins with a header: `# Codebase Context (Semantic Search)`
- **AND** each entry is formatted with name, kind, file, line, and snippet
- **AND** the total file size does not exceed 4,000 tokens

### Requirement: Codebase context hydration is skipped when CUDGEL_ENDPOINT is unset
When the `CUDGEL_ENDPOINT` environment variable is absent or empty in the hydration init-container, the `SeedCodebaseContext` step SHALL be skipped entirely with no error.

#### Scenario: No-op when endpoint not configured
- **WHEN** `CUDGEL_ENDPOINT` is not set in the hydrator environment
- **THEN** no HTTP call is made
- **AND** no `.aot/context/codebase.md` is written
- **AND** hydration completes normally
