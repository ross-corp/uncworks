# list-response-limits Specification

## Purpose
TBD - created by archiving change code-quality-hardening. Update Purpose after archive.
## Requirements
### Requirement: List endpoints cap results to prevent OOM
All server list endpoints SHALL return at most 500 items per response. When the underlying store contains more items, the response MUST be truncated at 500 with no error (truncation is silent to the caller for now).

#### Scenario: Small list returned in full
- **WHEN** the store contains 10 items
- **WHEN** a GET list request is made
- **THEN** all 10 items are returned

#### Scenario: Large list is truncated
- **WHEN** the store contains 1000 items
- **WHEN** a GET list request is made
- **THEN** exactly 500 items are returned
- **THEN** HTTP 200 is returned (not an error)

