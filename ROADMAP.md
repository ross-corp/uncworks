# AOT Roadmap

Items graduate from Future → Next → Current as they're prioritized. When an item
moves to Current, create an OpenSpec change (`/opsx:propose`) to formalize the
design and track implementation.

## Current

*No items currently in progress.*

## Next

### Knowledge System (Brain + Embeddings + Search)
The knowledge system code exists (`internal/brain`, `internal/embeddings`,
`SearchPastWork` API, `PersistRunData`/`EmbedRunData`/`HydrateContext` activities)
but is not initialized with real PostgreSQL or embedding provider connections.
Enabling it requires:
- PostgreSQL connection setup in the API server and Temporal worker
- Embedding provider configuration (Ollama or OpenAI embeddings endpoint)
- `BrainSearcher` and `Embedder` initialization in `NewAOTServiceHandler`
- `BrainStore` and `Embedder` initialization in `KnowledgeActivities`
- UI component for SearchPastWork (search bar + results display)

### Spec-Driven Pipeline Hardening
The Plan → Execute → Verify pipeline is wired but needs real-world testing:
- Direct exec in sidecar (replace `execInSidecar` agent-spawning with lightweight bash exec)
- Streaming verification progress to the frontend
- Verification result display in the UI for completed runs
- Automated spec scenario command extraction (parse WHEN/THEN for `npm test`, `go test`, etc.)

## Future

### Real-Time Log Streaming for Spec-Driven Stages
Stage transitions (plan → execute → verify) should stream live output to the
frontend, not just show results after each stage completes.

### User-Editable Specs Mid-Run
Allow users to pause a spec-driven run after the Plan stage, edit the generated
spec, and resume execution with the modified spec.

### Custom Verification Scripts
Allow users to provide their own verification commands beyond what's extracted
from spec scenarios (e.g., custom test suites, integration checks).

### Multi-Cluster Support
Run agents on remote Kubernetes clusters, not just the local k0s instance.

---

## Process

1. **Ideas** start in Future with a brief description
2. **Prioritized items** move to Next with enough detail to estimate scope
3. **Active work** moves to Current when an OpenSpec change is created
4. When a change is archived, its Current entry is removed
5. Anyone can propose additions — open a PR or discuss in an issue
