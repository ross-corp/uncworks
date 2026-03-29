#!/bin/sh
# build-frontend.sh — builds the web frontend and syncs the dist output into
# the wails embed directory (cmd/uncworks-app/frontend/).
# Wails runs frontend:build from the frontend/ subdirectory, so we resolve
# paths relative to that.
set -e
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
cd "$ROOT/web"
npm run build
rsync -a --delete dist/ "$ROOT/cmd/uncworks-app/frontend/dist/"
