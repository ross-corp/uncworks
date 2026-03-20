#!/usr/bin/env bash
#
# check-doc-staleness.sh — detect stale code references in documentation
#
# Extracts backtick-quoted identifiers from docs/*.md, checks whether they
# still exist in Go and TypeScript source files, and reports misses.
# Exits 1 if more than 5 stale references are found.
#
set -euo pipefail

DOCS_DIR="${1:-docs}"
STALE_COUNT=0
STALE_REFS=()

# Collect all backtick-quoted identifiers from docs
mapfile -t IDENTIFIERS < <(
  grep -rhoP '`[A-Za-z_][A-Za-z0-9_.#:]*(?:\(\))?`' "$DOCS_DIR"/*.md "$DOCS_DIR"/**/*.md 2>/dev/null \
    | sed 's/^`//;s/`$//' \
    | sort -u
)

# Filter to identifiers that look like code references:
#   - contains an uppercase letter (type/struct names, Go exports)
#   - contains a dot (package.Method, object.field)
#   - ends with () (function calls)
#   - contains :: or # (namespace refs)
# Exclude common non-code words and very short identifiers
filter_identifier() {
  local id="$1"

  # Skip very short identifiers (likely not code refs)
  [[ ${#id} -lt 3 ]] && return 1

  # Skip common markdown/doc terms that happen to match
  case "$id" in
    README|NOTE|TODO|IMPORTANT|WARNING|FIXME|TBD|WIP|URL|URI|API|CLI|TUI|SSH|DNS|TTL|LLM|RAM|CPU|GPU|PVC|CRD)
      return 1 ;;
  esac

  # Skip Helm dotted-notation values (e.g., web.port, worker.image, llm.baseUrl)
  # These are config keys, not code references.
  if [[ "$id" =~ ^[a-z]+\.[a-z] ]]; then
    return 1
  fi

  # Must look like a code reference
  if [[ "$id" =~ [A-Z][a-z] ]] ||       # CamelCase
     [[ "$id" =~ \. ]] ||                 # dot notation
     [[ "$id" =~ \(\)$ ]] ||              # function call
     [[ "$id" =~ :: ]] ||                 # namespace
     [[ "$id" =~ \# ]] ||                 # anchor/method
     [[ "$id" =~ ^[a-z]+[A-Z] ]]; then   # camelCase
    return 0
  fi

  return 1
}

echo "=== Doc Staleness Check ==="
echo "Scanning $DOCS_DIR for code references..."
echo ""

for id in "${IDENTIFIERS[@]}"; do
  filter_identifier "$id" || continue

  # Strip trailing () for searching
  search_id="${id%()}"

  # Search in Go and TypeScript source files (exclude vendor, node_modules, docs, generated)
  found=$(grep -rl --include='*.go' --include='*.ts' --include='*.tsx' \
    -F "$search_id" \
    . \
    --exclude-dir=vendor \
    --exclude-dir=node_modules \
    --exclude-dir=.git \
    --exclude-dir=docs \
    --exclude-dir='*.pb.go' \
    2>/dev/null | head -1 || true)

  if [ -z "$found" ]; then
    echo "  STALE: \`$id\` — not found in source"
    STALE_REFS+=("$id")
    STALE_COUNT=$((STALE_COUNT + 1))
  fi
done

echo ""
echo "=== Results ==="
echo "Total identifiers scanned: ${#IDENTIFIERS[@]}"
echo "Stale references found: $STALE_COUNT"

if [ "$STALE_COUNT" -gt 0 ]; then
  echo ""
  echo "Stale references:"
  for ref in "${STALE_REFS[@]}"; do
    echo "  - $ref"
  done
fi

if [ "$STALE_COUNT" -gt 5 ]; then
  echo ""
  echo "FAIL: More than 5 stale references detected. Please update documentation."
  exit 1
fi

echo ""
echo "OK: Staleness within acceptable threshold."
exit 0
