## ADDED Requirements

### Requirement: Server-side response shaping
The system SHALL transform backend responses in the BFF before returning to the frontend, moving transformation logic from client-side to server-side.

#### Scenario: Span name remapping
- **WHEN** the BFF returns trace spans with legacy names (unc.*, neph.*)
- **THEN** span names SHALL be remapped to manage.*/implement.* server-side

#### Scenario: Cost estimation
- **WHEN** trace spans contain token usage metadata
- **THEN** the BFF SHALL compute and include cost estimates using the pricing table

#### Scenario: Span deduplication
- **WHEN** spans.jsonl contains duplicate span IDs (open + close)
- **THEN** the BFF SHALL deduplicate server-side (already done in readSpansFile, but now explicitly a BFF responsibility)

### Requirement: Intelligent caching
The system SHALL cache API responses with TTLs based on run phase.

#### Scenario: Active run short cache
- **WHEN** a run is in RUNNING phase
- **THEN** trace and log responses SHALL be cached for 3 seconds

#### Scenario: Completed run long cache
- **WHEN** a run is in SUCCEEDED/FAILED/CANCELLED phase
- **THEN** trace, log, and file responses SHALL be cached for 5 minutes
