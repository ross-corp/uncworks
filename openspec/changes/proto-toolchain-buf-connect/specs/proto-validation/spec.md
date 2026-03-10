## ADDED Requirements

### Requirement: protovalidate import in proto files

Proto files SHALL import `buf/validate/validate.proto` to enable protovalidate annotations.

#### Scenario: proto files import protovalidate

- **WHEN** a developer inspects any `.proto` file that defines request or spec messages
- **THEN** the file contains `import "buf/validate/validate.proto";`

### Requirement: required fields have protovalidate annotations

Required fields in proto messages SHALL have protovalidate annotations that enforce non-empty or non-default constraints.

#### Scenario: required string field has non-empty rule

- **WHEN** a string field is semantically required (e.g., `prompt`)
- **THEN** the field has a `(buf.validate.field).string.min_len = 1` annotation or equivalent non-empty constraint

### Requirement: AgentRunSpec.repo_url validated as URI

`AgentRunSpec.repo_url` SHALL be validated as a valid URI using protovalidate annotations.

#### Scenario: valid repo_url accepted

- **WHEN** a client sends a `CreateAgentRun` request with `repo_url` set to `https://github.com/org/repo`
- **THEN** the request passes validation

#### Scenario: invalid repo_url rejected

- **WHEN** a client sends a `CreateAgentRun` request with `repo_url` set to `not-a-url`
- **THEN** the server returns an `INVALID_ARGUMENT` error code with a message indicating the URI format violation

### Requirement: AgentRunSpec.prompt validated as non-empty

`AgentRunSpec.prompt` SHALL be validated as non-empty using protovalidate annotations.

#### Scenario: empty prompt rejected

- **WHEN** a client sends a `CreateAgentRun` request with an empty `prompt` field
- **THEN** the server returns an `INVALID_ARGUMENT` error code with a message indicating the field must not be empty

### Requirement: AgentRunSpec.backend validated as not BACKEND_UNSPECIFIED

`AgentRunSpec.backend` SHALL be validated to ensure it is not set to `BACKEND_UNSPECIFIED` using protovalidate annotations.

#### Scenario: unspecified backend rejected

- **WHEN** a client sends a `CreateAgentRun` request with `backend` set to `BACKEND_UNSPECIFIED` (value 0)
- **THEN** the server returns an `INVALID_ARGUMENT` error code with a message indicating the backend must be specified

### Requirement: AgentRunSpec.ttl_seconds validated as positive when set

`AgentRunSpec.ttl_seconds` SHALL be validated as greater than 0 when the field is set, using protovalidate annotations.

#### Scenario: zero ttl_seconds rejected

- **WHEN** a client sends a `CreateAgentRun` request with `ttl_seconds` set to `0`
- **THEN** the server returns an `INVALID_ARGUMENT` error code with a message indicating the value must be greater than 0

#### Scenario: positive ttl_seconds accepted

- **WHEN** a client sends a `CreateAgentRun` request with `ttl_seconds` set to `3600`
- **THEN** the request passes validation for the `ttl_seconds` field

### Requirement: server-side validation via Connect interceptor

Server handlers SHALL enforce protovalidate rules via a Connect interceptor. The interceptor MUST validate incoming request messages before they reach the handler logic.

#### Scenario: interceptor runs before handler

- **WHEN** a client sends a request with an invalid field
- **THEN** the Connect interceptor rejects the request before the handler function is invoked

#### Scenario: interceptor passes valid requests

- **WHEN** a client sends a request with all fields passing protovalidate rules
- **THEN** the Connect interceptor passes the request through to the handler function

### Requirement: invalid requests return INVALID_ARGUMENT

Invalid requests SHALL return an `INVALID_ARGUMENT` error code with a descriptive message identifying which field(s) failed validation and why.

#### Scenario: error message identifies the field

- **WHEN** a client sends a request with `prompt` empty and `repo_url` malformed
- **THEN** the server returns an `INVALID_ARGUMENT` error whose message identifies both `prompt` and `repo_url` as invalid, with reasons for each
