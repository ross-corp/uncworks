# context-hydration Specification

## Purpose
TBD - created by archiving change persistent-knowledge-system. Update Purpose after archive.
## Requirements
### Requirement: New runs receive relevant context from past work
Before the agent starts, the system SHALL query pgvector for past work relevant to the current run's prompt and repository. Relevant results SHALL be written to the agent's workspace as a context file.

#### Scenario: Run with relevant past work
- **WHEN** a new agent run starts with a prompt and repo URL
- **AND** past runs have produced embeddings for the same or similar repos
- **THEN** the HydrateContext activity queries pgvector for the top-K relevant chunks
- **AND** a context file is written to `.aot/context/past-work.md` in the agent workspace
- **AND** the file contains formatted summaries of relevant past diffs, logs, and traces

#### Scenario: Run with no relevant past work
- **WHEN** a new agent run starts
- **AND** pgvector contains no relevant results above the similarity threshold
- **THEN** no context file is written
- **AND** the agent starts normally without past-work context

#### Scenario: First-ever run
- **WHEN** the system has no historical data (empty code_chunks and trace_chunks tables)
- **THEN** the HydrateContext activity completes immediately with no context file
- **AND** the agent starts normally

### Requirement: Context hydration runs as a Temporal activity before agent start
The `HydrateContext` activity SHALL execute in the workflow between workspace provisioning and agent startup. It SHALL have a 5-second timeout and degrade gracefully on failure.

#### Scenario: Hydration completes within timeout
- **WHEN** the HydrateContext activity runs
- **AND** pgvector responds within 5 seconds
- **THEN** the context file is written and the workflow proceeds to agent startup

#### Scenario: Hydration times out
- **WHEN** the HydrateContext activity exceeds the 5-second timeout
- **THEN** the activity is cancelled
- **AND** the workflow proceeds to agent startup without context
- **AND** a warning is logged

#### Scenario: Hydration fails with error
- **WHEN** the HydrateContext activity fails (database error, embedding error)
- **THEN** the error is logged
- **AND** the workflow proceeds to agent startup without context
- **AND** the run's final status is NOT affected by the hydration failure

### Requirement: Query uses prompt embedding and repo filter
The hydration query SHALL embed the run's prompt using the same ONNX model and search both code_chunks and trace_chunks. Results SHALL be filtered by repo URL when the run targets a specific repository.

#### Scenario: Query with repo filter
- **WHEN** the run targets repo `github.com/org/repo`
- **THEN** the pgvector query includes a WHERE clause filtering `repo_url` to that repo
- **AND** results from other repos are excluded

#### Scenario: Query without repo filter
- **WHEN** the run does not specify a repo URL (or targets multiple repos)
- **THEN** the pgvector query searches across all repos
- **AND** repo URL is included in result metadata for context

### Requirement: Context file is formatted for agent consumption
The context file SHALL be markdown formatted with clear sections, including the source run ID, similarity score, and chunk content. It SHALL be concise enough for the agent to process without exceeding context limits.

#### Scenario: Context file structure
- **WHEN** a context file is generated with results from past work
- **THEN** it contains a header explaining the context source
- **AND** each result includes: source run ID, timestamp, file path (for code) or activity name (for traces), and the chunk text
- **AND** total file size does not exceed 8,000 tokens

### Requirement: Senior agents receive richer context
When the current run is a senior/orchestrator agent, the hydration query SHALL increase the result limit (top-K) and include planning-level information from past decomposition decisions.

#### Scenario: Senior agent hydration
- **WHEN** the run's agent type is "senior" or "orchestrator"
- **THEN** the top-K limit is increased from 10 to 25
- **AND** results include trace chunks from past senior agent planning decisions

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

### Requirement: Broken clone directory is detected and recovered
The hydration system SHALL detect when a `.bare` clone directory exists but is not a valid git repository. When detected, it MUST remove the broken directory and reattempt the clone rather than proceeding with corrupt state.

#### Scenario: Broken bare directory is replaced
- **WHEN** a `.bare` clone directory exists from a previous failed clone
- **WHEN** `git rev-parse --git-dir` fails on that directory
- **THEN** the directory is removed
- **THEN** a fresh clone is performed

#### Scenario: Valid existing clone is reused
- **WHEN** a `.bare` clone directory exists and is a valid git repository
- **THEN** the clone step is skipped and the existing repo is used

### Requirement: Clone failure cleans up partial state
The hydration system SHALL remove any partially-created `.bare` directory when a `git clone` fails, so that subsequent retries do not encounter corrupt state.

#### Scenario: Clone failure removes partial directory
- **WHEN** `git clone` fails (network error, auth failure, etc.)
- **THEN** the partially-created `.bare` directory is removed
- **THEN** the error is returned to the caller

### Requirement: Malformed AOT_REPOS logs a warning
When the `AOT_REPOS` environment variable contains invalid JSON, the hydration system SHALL log a warning indicating the parse failure and fall back to single-repo mode. It MUST NOT silently ignore the malformed input.

#### Scenario: Invalid JSON triggers warning
- **WHEN** `AOT_REPOS` is set to `{invalid json}`
- **THEN** a warning is logged containing "failed to parse AOT_REPOS"
- **THEN** hydration falls back to single-repo mode using `AOT_REPO_URL`

