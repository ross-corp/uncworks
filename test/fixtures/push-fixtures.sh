#!/usr/bin/env bash
# Push fixture repositories to a Soft-Serve git server.
# Usage: SOFT_SERVE_ADDR=localhost:9418 ./push-fixtures.sh
set -euo pipefail

SOFT_SERVE_ADDR="${SOFT_SERVE_ADDR:-localhost:9418}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

push_fixture() {
  local name="$1"
  local src="$2"
  local repo_dir="$WORK_DIR/$name"

  echo "==> Pushing fixture: $name"
  mkdir -p "$repo_dir"
  cp -r "$src"/. "$repo_dir"/
  cd "$repo_dir"
  git init -q
  git add -A
  git commit -q -m "Initial commit"
  git remote add origin "git://${SOFT_SERVE_ADDR}/${name}"
  git push -q origin HEAD:main 2>/dev/null || {
    # Soft-serve auto-creates repos on push via SSH; for git:// we may need SSH push
    # Fall back to SSH push if git:// push fails
    git remote set-url origin "ssh://localhost:23231/${name}"
    git push -q origin HEAD:main
  }
  echo "    Pushed $name to git://${SOFT_SERVE_ADDR}/${name}"
}

push_fixture "e2e-repo" "$SCRIPT_DIR/e2e-repo"
push_fixture "e2e-repo-frontend" "$SCRIPT_DIR/e2e-repo-frontend"

echo "==> All fixtures pushed successfully"
