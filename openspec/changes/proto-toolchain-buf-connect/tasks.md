## 1. Buf Toolchain Setup

- [x] 1.1 Add `buf` to `devbox.json`, remove `protoc-gen-go` and `protoc-gen-go-grpc` (buf manages plugins)
- [x] 1.2 Create `buf.yaml` at project root: v2 format, module path `proto/`, lint rules DEFAULT, breaking rules FILE
- [x] 1.3 Create `buf.gen.yaml`: Go plugins (protoc-gen-go, protoc-gen-go-grpc) outputting to `gen/go/`, TypeScript plugins (protoc-gen-es, protoc-gen-connect-es) outputting to `gen/ts/`
- [x] 1.4 Run `buf generate` and verify Go output in `gen/go/` matches current generated code
- [x] 1.5 Run `buf generate` and verify TypeScript output in `gen/ts/` produces valid Connect client stubs
- [x] 1.6 Run `buf lint` and fix any violations in `proto/api.proto` and `proto/agent.proto`
- [x] 1.7 Delete `hack/proto-gen.sh`
- [x] 1.8 Update `Taskfile.yml`: replace `proto:gen` with `buf generate`, add `proto:lint` and `proto:breaking` targets
- [x] 1.9 Commit `buf.lock` to version control

## 2. Protovalidate Annotations

- [x] 2.1 Add `buf.build/bufbuild/protovalidate` dependency to `buf.yaml`
- [x] 2.2 Add protovalidate annotations to `proto/api.proto`: `AgentRunSpec.repo_url` (URI), `prompt` (non-empty), `backend` (not UNSPECIFIED), `ttl_seconds` (> 0 when set)
- [x] 2.3 Add protovalidate annotations to `proto/agent.proto`: `StartAgentRequest.agent_run_id` (non-empty), `prompt` (non-empty)
- [x] 2.4 Run `buf generate` to regenerate code with validation descriptors
- [x] 2.5 Add `bufbuild/protovalidate-go` to `go.mod` for server-side validation

## 3. ConnectRPC Server

- [x] 3.1 Add `connectrpc.com/connect` and `connectrpc.com/grpcreflect` to `go.mod`
- [x] 3.2 Add `connectrpc.com/validate` (protovalidate interceptor) to `go.mod`
- [x] 3.3 Rewrite `internal/server/grpc.go` as ConnectRPC handlers: implement `AOTService` using `connect.NewUnaryHandler` / `connect.NewServerStreamHandler`
- [x] 3.4 Add protovalidate Connect interceptor to reject invalid requests with INVALID_ARGUMENT
- [x] 3.5 Update `cmd/apiserver/main.go`: single `net/http` server on `:50051` serving gRPC + Connect + gRPC-Web protocols via `connectrpc.com/grpchealth` and handler mux
- [x] 3.6 Rewrite `internal/sidecar/gateway.go` as ConnectRPC handlers for `AgentSidecarService` and `AgentNotificationService`
- [x] 3.7 Update `cmd/sidecar/main.go` to use ConnectRPC server on `:50052`
- [x] 3.8 Delete `internal/server/websocket.go`
- [x] 3.9 Remove `:8080` HTTP server and `/ws` endpoint from `cmd/apiserver/main.go`
- [x] 3.10 Update all Go unit tests (`internal/server/grpc_test.go`) to test ConnectRPC handlers

## 4. TypeScript Connect Client

- [ ] 4.1 Add `@connectrpc/connect`, `@connectrpc/connect-web`, `@connectrpc/protoc-gen-connect-es`, `@bufbuild/protobuf`, `@bufbuild/protoc-gen-es` to `packages/shared/package.json`
- [ ] 4.2 Remove `@grpc/grpc-js` and `@grpc/proto-loader` from `packages/shared/package.json`
- [ ] 4.3 Replace `packages/shared/src/grpc/client.ts` with a Connect client wrapper that uses the generated `gen/ts/` stubs
- [ ] 4.4 Update `packages/shared/` exports: `./grpc` module now re-exports the Connect client
- [ ] 4.5 Update `web/` to use Connect streaming for `WatchAgentRun` (replace WebSocket connection logic in dashboard components)
- [ ] 4.6 Remove WebSocket client code from `web/src/` (any `new WebSocket()` calls, reconnect logic)
- [ ] 4.7 Update `packages/pi-aot-extension/` if it uses the shared gRPC client
- [ ] 4.8 Update `packages/tui/` if it uses the shared gRPC client
- [ ] 4.9 Run all TypeScript tests and fix any breakages

## 5. Mermaid Documentation

- [ ] 5.1 Convert README.md architecture diagram (lines 11-48) to Mermaid `graph TD`
- [ ] 5.2 Add Mermaid AgentRun lifecycle state diagram (`stateDiagram-v2`) to README.md or docs/user-guide.md
- [ ] 5.3 Add Mermaid sequence diagram for HITL flow to docs/user-guide.md
- [ ] 5.4 Add Mermaid sequence diagram for multi-agent (spawn_junior) flow to docs/user-guide.md
- [ ] 5.5 Convert any ASCII diagrams in `openspec/changes/*/design.md` and `openspec/changes/*/proposal.md` to Mermaid
- [ ] 5.6 Add Mermaid data flow diagram (client → API → controller → pod → sidecar → brain) to docs/user-guide.md

## 6. Verification

- [ ] 6.1 Run `buf lint` -- zero violations
- [ ] 6.2 Run `buf breaking --against '.git#branch=main'` -- passes (no breaking changes from this PR since we're regenerating)
- [ ] 6.3 Run `task test:go` -- all Go tests pass with ConnectRPC handlers
- [ ] 6.4 Run `task test:web` -- Playwright tests pass with Connect streaming
- [ ] 6.5 Run `task test:shared` -- shared package tests pass with Connect client
- [ ] 6.6 Verify gRPC clients (grpcurl) still work against the ConnectRPC server (wire compatibility)
- [ ] 6.7 Verify web dashboard streams agent events via Connect (no WebSocket)
