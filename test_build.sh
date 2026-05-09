#!/bin/bash
# Try to build the server package
cd /workspace/uncworks
echo "Checking if we can find go..."
find / -name "go" -type f -executable 2>/dev/null | head -5
echo "Trying to build..."
if command -v go >/dev/null 2>&1; then
    go build ./internal/server
else
    echo "go not found in PATH"
fi