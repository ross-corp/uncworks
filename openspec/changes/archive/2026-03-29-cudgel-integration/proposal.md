## Why

Agents currently have no awareness of the codebase they are operating on until they manually grep or read files. Cudgel is a Rust-based semantic code search engine (TreeSitter symbol extraction + pgvector embeddings) that can provide agents a structured "map" of relevant code before they start and on demand during execution — eliminating redundant discovery work across runs and letting agents leverage institutional knowledge encoded in the codebase itself.

## What Changes

- **New**: `cudgel` deployed as a Kubernetes service in the aot cluster, backed by a thin HTTP shim wrapping the CLI, using the existing postgres instance (adding pgvector extension) for vector storage
- **New**: `internal/cudgel/` Go client package with `SemanticSearch` and `GraphTraversal` methods
- **New**: `cudgel-agent-tool` capability — `semantic_search` exposed as a callable tool for agents during run execution, replacing manual grep/find patterns
- **Modified**: `context-hydration` — hydration at run start now also calls cudgel for source code symbols (in addition to existing past-work pgvector search), injecting a codebase map into the initial context
- **Modified**: `semantic-search-api` — extends existing SearchPastWork to also delegate to cudgel for source code queries, unifying the search interface
- Helm chart (`deploy/helm/aot/`) updated with cudgel Deployment, Service, and ConfigMap
- No changes to OpenSpec declarations, Temporal workflow topology, or agent role separation

## Capabilities

### New Capabilities

- `cudgel-service`: Cudgel binary deployed as a k8s Deployment with a thin HTTP shim exposing `/search`, `/graph`, and `/index` endpoints; backed by postgres+pgvector; Go client in `internal/cudgel/`
- `cudgel-agent-tool`: `semantic_search(query string) → []CodeChunk` exposed as a sidecar tool callable by agents during execution, registered in the gateway alongside existing file/exec tools

### Modified Capabilities

- `context-hydration`: At run start, hydration now also seeds codebase symbols from cudgel (in addition to past-work embeddings), writing source context to `.aot/context/codebase.md` alongside the existing `past-work.md`
- `semantic-search-api`: `SearchPastWork` extended with a `source_filter = SOURCE_CODE` mode that queries cudgel rather than the internal code_chunks table, returning live symbol-level results

## Impact

- **New k8s dependency**: cudgel Deployment + HTTP shim container in aot cluster; postgres pgvector extension must be enabled (non-breaking if postgres is already present)
- **`internal/cudgel/`**: New Go package, no existing code modified
- **`internal/hydration/`**: Hydration activity gains a second query path; existing past-work path unchanged
- **`internal/sidecar/gateway.go`**: One new tool registration; no existing tools modified
- **`deploy/helm/aot/`**: New templates for cudgel Deployment/Service; values.yaml gains `cudgel.enabled` and `cudgel.endpoint` fields
- **No API surface changes** to existing gRPC services; `SearchPastWork` filter enum gains one value (additive, non-breaking)
