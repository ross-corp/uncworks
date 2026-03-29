## ADDED Requirements

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
