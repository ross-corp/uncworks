## ADDED Requirements

### Requirement: Cudgel HTTP shim exposes semantic search endpoint
The cudgel-shim service SHALL expose a `POST /search` endpoint that accepts a JSON body with `query` (string) and `limit` (int, default 10, max 50) fields and returns a JSON array of symbol results. Each result SHALL include `name`, `kind`, `file`, `line`, `snippet`, and `score` fields.

#### Scenario: Successful search returns ranked symbols
- **WHEN** a client POSTs `{"query": "authentication middleware", "limit": 5}` to `/search`
- **THEN** the response is HTTP 200 with a JSON array of up to 5 symbols
- **AND** each symbol contains `name`, `kind`, `file`, `line`, `snippet`, and `score`
- **AND** results are ordered by descending similarity score

#### Scenario: Search with empty query returns 400
- **WHEN** a client POSTs `{"query": ""}` to `/search`
- **THEN** the response is HTTP 400 with an error message

#### Scenario: Cudgel binary unavailable returns 503
- **WHEN** the cudgel binary is not found or fails to execute
- **THEN** the response is HTTP 503 with an error body
- **AND** the shim logs the error

### Requirement: Cudgel HTTP shim exposes graph traversal endpoint
The cudgel-shim service SHALL expose a `POST /graph` endpoint that accepts `symbol` (string) and `depth` (int, default 2, max 5) and returns a JSON array of edges. Each edge SHALL include `from`, `to`, and `kind` fields.

#### Scenario: Graph traversal returns call graph edges
- **WHEN** a client POSTs `{"symbol": "internal/brain.Store.Search", "depth": 2}` to `/graph`
- **THEN** the response is HTTP 200 with a JSON array of edges
- **AND** each edge includes `from`, `to`, and `kind`

#### Scenario: Unknown symbol returns empty graph
- **WHEN** a client POSTs with a symbol that does not exist in the index
- **THEN** the response is HTTP 200 with an empty array

### Requirement: Cudgel HTTP shim exposes index trigger endpoint
The cudgel-shim service SHALL expose a `POST /index` endpoint that accepts `repo_path` (string) and triggers `cudgel index <repo_path>` asynchronously. The endpoint SHALL return immediately with HTTP 202 Accepted.

#### Scenario: Index request is accepted and runs asynchronously
- **WHEN** a client POSTs `{"repo_path": "/workspace/myrepo"}` to `/index`
- **THEN** the response is HTTP 202 Accepted immediately
- **AND** cudgel index runs in the background
- **AND** index completion or failure is logged

#### Scenario: Index request with absolute path outside allowed prefix is rejected
- **WHEN** a client POSTs `{"repo_path": "/etc/passwd"}` to `/index`
- **THEN** the response is HTTP 400 Bad Request

### Requirement: Cudgel service is deployed as a k8s Deployment in the aot cluster
The cudgel-shim SHALL be deployed as a Kubernetes Deployment with a corresponding Service resource. The Deployment SHALL use a dedicated container image (`cudgel-shim`) that bundles both the Go HTTP shim binary and the cudgel Rust binary. The Service SHALL be reachable at `cudgel.aot.svc.cluster.local:8080` (or configurable via helm values).

#### Scenario: Service is reachable from agent pods
- **WHEN** an agent pod performs an HTTP request to `http://cudgel.aot.svc.cluster.local:8080/search`
- **THEN** the request is routed to the cudgel-shim Deployment

#### Scenario: Deployment can be disabled via helm values
- **WHEN** `cudgel.enabled` is set to `false` in helm values
- **THEN** no cudgel Deployment or Service resources are created

### Requirement: Cudgel uses the existing postgres instance with a dedicated database
The cudgel-shim SHALL connect to the existing cluster postgres instance using a `cudgel` database. The database connection string SHALL be provided via a k8s Secret mounted as the `CUDGEL_DATABASE_URL` environment variable. Cudgel SHALL NOT share tables with any other application database.

#### Scenario: Cudgel database is isolated from other application databases
- **WHEN** cudgel writes index data
- **THEN** it writes only to tables within the `cudgel` database
- **AND** no tables in the `aot` or `litellm` databases are affected
