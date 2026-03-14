## ADDED Requirements

### Requirement: Browse agent pod filesystem via REST API
The API server SHALL expose REST endpoints to list directories and read file contents from running or retained agent pods.

#### Scenario: List directory contents
- **WHEN** a GET request is made to `/api/v1/runs/{id}/files?path=/workspace/src`
- **THEN** the response contains a JSON array of entries with name, type (file/directory), size, and modification time

#### Scenario: Read file content
- **WHEN** a GET request is made to `/api/v1/runs/{id}/files/content?path=/workspace/src/repo/main.go`
- **THEN** the response contains the raw file content

#### Scenario: Pod not available
- **WHEN** a file request is made for a run whose pod no longer exists
- **THEN** the API returns HTTP 404 with a clear error message indicating the pod has been deleted

### Requirement: File tree component in web UI
The detail panel Files tab SHALL display a navigable directory tree with file preview.

#### Scenario: Directory tree renders workspace
- **WHEN** the user opens the Files tab for a running or retained run
- **THEN** a tree view shows the `/workspace` directory structure
- **AND** directories are expandable (lazy-loaded on click)

#### Scenario: File preview in Monaco editor
- **WHEN** the user clicks a file in the tree
- **THEN** the file content loads in a read-only Monaco editor with syntax highlighting
- **AND** the editor language mode is auto-detected from the file extension
