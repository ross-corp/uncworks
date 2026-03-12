## ADDED Requirements

### Requirement: buf.yaml module configuration

buf.yaml SHALL exist at the project root and define the proto module using buf v2 config format. It SHALL configure lint rules to the `DEFAULT` preset and breaking change rules to the `FILE` preset.

#### Scenario: buf.yaml is present and valid

- **WHEN** a developer runs `buf config ls-lint-rules` from the project root
- **THEN** the DEFAULT lint rule set is active and the command exits successfully

#### Scenario: buf.yaml enforces FILE-level breaking detection

- **WHEN** a developer runs `buf config ls-breaking-rules` from the project root
- **THEN** the FILE breaking rule set is active

### Requirement: buf.gen.yaml code generation configuration

buf.gen.yaml SHALL define code generation plugins for both Go and TypeScript targets. Go plugins SHALL include `protoc-gen-go` and `protoc-gen-go-grpc`. TypeScript plugins SHALL include `protoc-gen-es` and `protoc-gen-connect-es`.

#### Scenario: buf.gen.yaml generates Go code

- **WHEN** `buf generate` is run
- **THEN** Go protobuf and gRPC stubs are written to the `gen/go/` directory

#### Scenario: buf.gen.yaml generates TypeScript code

- **WHEN** `buf generate` is run
- **THEN** TypeScript protobuf and Connect client stubs are written to the `gen/ts/` directory

### Requirement: task proto:gen runs buf generate

`task proto:gen` SHALL invoke `buf generate` to produce all generated code from proto source files.

#### Scenario: proto:gen produces up-to-date stubs

- **WHEN** a developer modifies a `.proto` file and runs `task proto:gen`
- **THEN** the `gen/go/` and `gen/ts/` directories contain regenerated stubs reflecting the proto changes

### Requirement: task proto:lint runs buf lint

`task proto:lint` SHALL invoke `buf lint` and SHALL fail with a non-zero exit code if any lint violation is detected.

#### Scenario: lint passes on well-formed protos

- **WHEN** all `.proto` files conform to the DEFAULT lint rules and a developer runs `task proto:lint`
- **THEN** the command exits with code 0 and produces no error output

#### Scenario: lint fails on violation

- **WHEN** a `.proto` file violates a DEFAULT lint rule (e.g., missing package declaration) and a developer runs `task proto:lint`
- **THEN** the command exits with a non-zero code and prints the violation details to stderr

### Requirement: task proto:breaking detects breaking changes

`task proto:breaking` SHALL invoke `buf breaking --against '.git#branch=main'` and SHALL fail with a non-zero exit code if any breaking change is detected relative to the main branch.

#### Scenario: no breaking changes

- **WHEN** a developer's branch has only additive proto changes and runs `task proto:breaking`
- **THEN** the command exits with code 0

#### Scenario: breaking change detected

- **WHEN** a developer's branch removes a field from a message and runs `task proto:breaking`
- **THEN** the command exits with a non-zero code and prints the breaking change details

### Requirement: buf.lock committed for reproducible builds

buf.lock SHALL be committed to version control. It MUST be updated whenever buf.yaml dependencies change.

#### Scenario: buf.lock is present in repository

- **WHEN** a developer clones the repository
- **THEN** `buf.lock` exists at the project root and `buf generate` produces deterministic output without fetching remote dependencies

### Requirement: Go generated code output directory

Generated Go code SHALL be written to the `gen/go/` directory within the project root.

#### Scenario: Go stubs are in gen/go/

- **WHEN** `buf generate` completes successfully
- **THEN** Go `.pb.go` and `_grpc.pb.go` files exist under `gen/go/` with correct Go package paths

### Requirement: TypeScript generated code output directory

Generated TypeScript code SHALL be written to the `gen/ts/` directory within the project root.

#### Scenario: TypeScript stubs are in gen/ts/

- **WHEN** `buf generate` completes successfully
- **THEN** TypeScript `.ts` files for protobuf messages and Connect service clients exist under `gen/ts/`
