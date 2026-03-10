## Why

AOT's proto toolchain uses raw `protoc` invocations via shell scripts (`hack/proto-gen.sh`) with no linting, no breaking change detection, and no TypeScript code generation from the same source. The web dashboard uses a hand-rolled WebSocket hub alongside the gRPC server, creating two separate API surfaces that drift independently. Adopting `buf` as the proto toolchain and ConnectRPC as the unified transport collapses these into a single, schema-enforced API that all clients (Go, TypeScript, browser) consume identically.

## What Changes

- **Replace `protoc` with `buf`**: Introduce `buf.yaml`, `buf.gen.yaml`, and `buf.lock` to manage proto linting, breaking change detection, and code generation declaratively.
- **Add `protovalidate` annotations**: Embed semantic validation rules (required fields, URI format, value bounds) directly in `.proto` files, eliminating hand-written validation in Go server code.
- **Adopt ConnectRPC server**: Replace `google.golang.org/grpc` server with `connectrpc.com/connect`, which serves gRPC, gRPC-Web, and Connect protocols on a single port.
- **Generate TypeScript clients from proto**: Use `@connectrpc/protoc-gen-connect-es` to generate typed TypeScript clients from the same `.proto` files, replacing the hand-written `@aot/shared/grpc/client.ts`.
- **Delete WebSocket hub**: Remove `internal/server/websocket.go` and the HTTP `:8080` endpoint. **BREAKING**: Web clients migrate from WebSocket to Connect server-streaming via `WatchAgentRun`.
- **Delete `hack/proto-gen.sh`**: Replaced by `buf generate`.
- **Convert all documentation diagrams to Mermaid**: Replace ASCII art diagrams in `README.md`, `docs/`, and `openspec/` with Mermaid diagrams. Never use ASCII box-drawing in markdown again.

## Capabilities

### New Capabilities
- `buf-proto-toolchain`: Declarative proto management with buf -- linting, breaking change detection, and multi-language code generation from a single `buf.gen.yaml`.
- `connect-transport`: Unified ConnectRPC server that replaces the dual gRPC + WebSocket architecture, enabling browser-native streaming without WebSocket.
- `proto-validation`: Protovalidate annotations for compile-time schema validation rules, enforced at both server and client boundaries.
- `mermaid-docs`: All architecture and flow diagrams in documentation use Mermaid syntax exclusively.

### Modified Capabilities
- `client-interfaces`: Web dashboard migrates from WebSocket client to ConnectRPC streaming client. TUI and CLI may optionally migrate to Connect client or remain on standard gRPC.
- `agent-harness`: Sidecar gRPC communication unchanged in protocol, but server implementation moves to ConnectRPC handler.

## Impact

- **`internal/server/`**: `grpc.go` rewritten as ConnectRPC handlers. `websocket.go` deleted. HTTP server on `:8080` removed.
- **`cmd/apiserver/`**: Single server on `:50051` serving gRPC + Connect + gRPC-Web.
- **`proto/`**: Both `.proto` files gain `protovalidate` imports and field annotations.
- **`gen/`**: Output restructured for buf -- Go stubs in `gen/go/`, TypeScript stubs in `gen/ts/`.
- **`packages/shared/`**: `grpc/client.ts` replaced with generated Connect client.
- **`web/`**: WebSocket connection logic replaced with Connect streaming.
- **`hack/proto-gen.sh`**: Deleted.
- **`Taskfile.yml`**: `proto:gen` updated to `buf generate`. New targets: `proto:lint`, `proto:breaking`.
- **`devbox.json`**: Add `buf` package. Remove `protoc-gen-go`, `protoc-gen-go-grpc` (buf manages plugins).
- **`README.md`, `docs/`, `openspec/`**: All ASCII diagrams converted to Mermaid.
- **Dependencies**: Add `connectrpc.com/connect`, `buf.build/gen/go/bufbuild/protovalidate`. Remove direct `protoc` dependency.
