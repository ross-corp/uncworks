## Context

The workspace layout changed from a single `/workspace/src/` directory to `/workspace/<repo>/` during the multi-repo hydration work. Several comments and spec documents were not updated. The doc staleness script has a regex that matches dotted Helm value references like `web.port` as if they were stale file paths.

## Goals / Non-Goals

**Goals:**
- Correct all `/workspace/src/` references to `/workspace/<repo>/`
- Eliminate false positives from the staleness script for Helm values
- Ensure no stale path references remain in comments, specs, or docs

**Non-Goals:**
- Rewriting the staleness script from scratch
- Updating any runtime code paths (they already use the correct layout)

## Decisions

### Decision 1: Simple find-and-replace for path comments

The path references are in comments and documentation only. A direct replacement of `/workspace/src/` with `/workspace/<repo>/` is sufficient since the surrounding context makes the new path clear.

### Decision 2: Allowlist pattern for staleness script

Add a regex exclusion for patterns matching `word.word` (Helm dotted notation) rather than trying to distinguish file paths from config keys heuristically.

## Risks / Trade-offs

- **Low risk**: All changes are in comments, docs, and a dev-only script. No runtime behavior affected.
