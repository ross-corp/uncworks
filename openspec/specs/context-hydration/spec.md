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

