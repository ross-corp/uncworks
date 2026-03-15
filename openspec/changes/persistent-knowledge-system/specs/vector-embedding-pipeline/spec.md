## ADDED Requirements

### Requirement: Code diffs are chunked using tree-sitter
After a run completes, file diffs SHALL be parsed with tree-sitter to produce code-aware chunks at function, method, or class boundaries. Each chunk SHALL retain its AST node type, file path, and language.

#### Scenario: Diff with multiple functions is split into chunks
- **WHEN** a diff modifies two functions in a Go file
- **THEN** two separate code chunks are created, one per function
- **AND** each chunk has `node_type` set to "function"
- **AND** each chunk has `file_path` and `language` set correctly

#### Scenario: Diff in unsupported language falls back to block chunking
- **WHEN** a diff modifies a file in a language without tree-sitter grammar
- **THEN** the diff is chunked by paragraph/block boundaries (lines of context)
- **AND** `node_type` is set to "block"

### Requirement: Log segments are chunked by paragraph
Non-code content (logs, trace descriptions, error messages) SHALL be chunked by paragraph boundaries with configurable overlap. Each chunk SHALL be at most 512 tokens.

#### Scenario: Long log output is split into overlapping chunks
- **WHEN** a run produces log output exceeding 512 tokens
- **THEN** it is split into chunks of at most 512 tokens
- **AND** adjacent chunks overlap by 64 tokens for context continuity

### Requirement: Chunks are embedded with all-MiniLM-L6-v2 via ONNX
All chunks SHALL be embedded using the sentence-transformers/all-MiniLM-L6-v2 model running locally via ONNX Runtime. The resulting vectors SHALL be 384-dimensional.

#### Scenario: Code chunk is embedded
- **WHEN** a code chunk is processed by the embedding pipeline
- **THEN** a 384-dimensional float32 vector is produced
- **AND** the vector is stored in the `code_chunks` table alongside the chunk text and metadata

#### Scenario: Trace chunk is embedded
- **WHEN** a trace/log chunk is processed by the embedding pipeline
- **THEN** a 384-dimensional float32 vector is produced
- **AND** the vector is stored in the `trace_chunks` table alongside the chunk text and metadata

### Requirement: Embeddings are stored in pgvector with HNSW index
Embedding vectors SHALL be stored in pgvector `vector(384)` columns. HNSW indexes SHALL be created with `m=16, ef_construction=200` using cosine distance for approximate nearest-neighbor search.

#### Scenario: Code chunks are searchable by vector similarity
- **WHEN** code chunks have been embedded and indexed
- **AND** a query vector is provided
- **THEN** the top-K most similar code chunks are returned ranked by cosine similarity
- **AND** query latency is under 100ms for up to 100,000 chunks

#### Scenario: Trace chunks are searchable by vector similarity
- **WHEN** trace chunks have been embedded and indexed
- **AND** a query vector is provided
- **THEN** the top-K most similar trace chunks are returned ranked by cosine similarity

### Requirement: Structural boosting weights code chunks by AST significance
Code chunks SHALL receive a `boost` score based on their AST node type. The boost is used as a multiplier when ranking search results.

#### Scenario: Function-level chunk ranks higher than whitespace change
- **GIVEN** a function-level chunk (boost 1.0) and a whitespace-only chunk (boost 0.1) with equal cosine similarity to a query
- **WHEN** search results are ranked
- **THEN** the function-level chunk appears first

#### Scenario: Boost values by node type
- **THEN** function/method definitions have boost 1.0
- **AND** class/struct definitions have boost 0.9
- **AND** import/require statements have boost 0.3
- **AND** whitespace-only changes have boost 0.1
- **AND** all other node types have boost 0.7

### Requirement: Dual index separates code from traces
Code chunks and trace chunks SHALL be stored in separate tables (`code_chunks` and `trace_chunks`) with independent HNSW indexes. Search queries SHALL be able to target one or both indexes.

#### Scenario: Code-only search
- **WHEN** a search query specifies `source_filter = "code"`
- **THEN** only `code_chunks` are searched
- **AND** trace chunks are not included in results

#### Scenario: Combined search
- **WHEN** a search query specifies no source filter
- **THEN** both `code_chunks` and `trace_chunks` are searched
- **AND** results are merged and re-ranked by boosted similarity score

### Requirement: Embedding pipeline runs asynchronously after run completion
The embedding pipeline SHALL run as a Temporal activity after the run completes. It SHALL NOT block the workflow completion signal or delay the run's terminal status.

#### Scenario: Embedding failure does not affect run status
- **WHEN** an agent run completes successfully
- **AND** the embedding pipeline fails (e.g., ONNX model error)
- **THEN** the run's final status remains Succeeded
- **AND** raw data in `run_logs`, `run_diffs`, `run_spans` is preserved
- **AND** embedding can be retried later
