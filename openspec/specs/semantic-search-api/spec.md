# semantic-search-api Specification

## Purpose
TBD - created by archiving change persistent-knowledge-system. Update Purpose after archive.
## Requirements
### Requirement: SearchPastWork gRPC endpoint accepts natural language queries
The AOT API service SHALL expose a `SearchPastWork` RPC that accepts a natural language query string and returns semantically relevant results from the knowledge base.

#### Scenario: Successful search with results
- **WHEN** a client calls `SearchPastWork` with query "authentication bug fix"
- **AND** the knowledge base contains relevant code chunks
- **THEN** the response contains ranked results with chunk text, source type, similarity score, run ID, and metadata
- **AND** results are ordered by relevance (boosted cosine similarity, descending)

#### Scenario: Search with no results
- **WHEN** a client calls `SearchPastWork` with a query that has no relevant matches
- **THEN** the response contains an empty results list
- **AND** no error is returned

#### Scenario: Search with empty query
- **WHEN** a client calls `SearchPastWork` with an empty query string
- **THEN** the API returns an InvalidArgument error

### Requirement: Search supports filtering by repository
The `SearchPastWork` request SHALL accept an optional `repo_url` filter. When provided, only chunks from runs targeting that repository are searched.

#### Scenario: Filtered search
- **WHEN** a client calls `SearchPastWork` with query "fix tests" and repo_url "github.com/org/repo"
- **THEN** only results from that repository are returned

### Requirement: Search supports filtering by time range
The `SearchPastWork` request SHALL accept optional `created_after` and `created_before` timestamp filters. When provided, only chunks from runs within that time range are searched.

#### Scenario: Time-filtered search
- **WHEN** a client calls `SearchPastWork` with a `created_after` timestamp of 7 days ago
- **THEN** only results from the last 7 days are returned

### Requirement: Search supports source type filter
The `SearchPastWork` request SHALL accept an optional `source_filter` enum (CODE, TRACE, ALL). When set to CODE, only code_chunks are searched. When set to TRACE, only trace_chunks. Default is ALL.

#### Scenario: Code-only search
- **WHEN** a client calls `SearchPastWork` with source_filter CODE
- **THEN** only code chunk results are returned (no trace/log results)

### Requirement: Search results include metadata
Each search result SHALL include: chunk text, source type (code or trace), similarity score, run ID, file path (for code chunks), language (for code chunks), node type (for code chunks), chunk type (for trace chunks), and creation timestamp.

#### Scenario: Code result metadata
- **WHEN** a search returns a code chunk result
- **THEN** the result includes `file_path`, `language`, `node_type`, `repo_url`, `run_id`, `similarity_score`, and `chunk_text`

#### Scenario: Trace result metadata
- **WHEN** a search returns a trace chunk result
- **THEN** the result includes `chunk_type`, `severity`, `repo_url`, `run_id`, `similarity_score`, and `chunk_text`

### Requirement: Search result limit is configurable
The `SearchPastWork` request SHALL accept a `limit` parameter (default 10, max 100). The API SHALL return at most `limit` results.

#### Scenario: Custom limit
- **WHEN** a client calls `SearchPastWork` with limit 5
- **THEN** at most 5 results are returned

#### Scenario: Limit exceeds maximum
- **WHEN** a client calls `SearchPastWork` with limit 200
- **THEN** at most 100 results are returned (clamped to max)

### Requirement: Search merges results from both indexes
When searching both code_chunks and trace_chunks, the API SHALL run parallel queries against both tables, merge results by boosted similarity score, and return the unified top-N.

#### Scenario: Merged ranking
- **WHEN** a combined search returns results from both indexes
- **THEN** code and trace results are interleaved by relevance
- **AND** a code chunk with boost 1.0 and similarity 0.8 ranks above a trace chunk with similarity 0.85 (effective score: 0.8 * 1.0 = 0.8 vs 0.85 * 1.0 = 0.85 -- trace wins, but a boosted code chunk at 0.9 similarity and 1.0 boost = 0.9 would win)

### Requirement: SearchPastWork accepts SOURCE_CODE filter to query cudgel
The `SearchPastWork` RPC SHALL accept a new `source_filter` enum value `SOURCE_CODE`. When `source_filter` is `SOURCE_CODE`, the API SHALL forward the query to the cudgel `/search` endpoint and return results from the cudgel index rather than the internal `code_chunks` table.

#### Scenario: SOURCE_CODE search returns symbols from cudgel
- **WHEN** a client calls `SearchPastWork` with query "database connection pooling" and `source_filter = SOURCE_CODE`
- **THEN** the API forwards the query to the cudgel service
- **AND** returns results with `source_type = SOURCE_CODE`
- **AND** each result includes `file_path`, `node_type` (mapped from cudgel `kind`), `similarity_score`, and `chunk_text` (mapped from cudgel `snippet`)

#### Scenario: SOURCE_CODE search when cudgel is unavailable returns empty results
- **WHEN** a client calls `SearchPastWork` with `source_filter = SOURCE_CODE`
- **AND** the cudgel service is unreachable
- **THEN** the response contains an empty results list
- **AND** no error is returned to the caller
- **AND** the failure is logged at WARN level

#### Scenario: Existing source filters are unaffected
- **WHEN** a client calls `SearchPastWork` with `source_filter = CODE` or `source_filter = TRACE`
- **THEN** the existing internal code_chunks and trace_chunks search behavior is unchanged
- **AND** no requests are made to cudgel

