#!/usr/bin/env bash
# proto-gen.sh — Generate Go code from protobuf definitions.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT}/proto"
GEN_DIR="${ROOT}/gen/go"

export PATH="${HOME}/go/bin:${PATH}"

echo "==> Generating Go code from protos..."

for proto in "${PROTO_DIR}"/*.proto; do
  echo "    $(basename "$proto")"
  protoc \
    --proto_path="${PROTO_DIR}" \
    --proto_path=/usr/local/include \
    --go_out="${GEN_DIR}" \
    --go_opt=module=github.com/uncworks/aot/gen/go \
    --go-grpc_out="${GEN_DIR}" \
    --go-grpc_opt=module=github.com/uncworks/aot/gen/go \
    "$proto"
done

echo "==> Done."
