## Why

The platform has evolved rapidly through multiple changes (agent-architecture-v2, docs-rebrand, CI fixes). Code has been added, renamed, and restructured without a systematic review. We need a comprehensive audit to identify: dead code, broken paths, stale tests, inconsistencies between components, missing error handling, and areas that need new proposals for improvement.

This is a meta-change: the output is not code fixes, but a list of issues and follow-up proposals.

## What Changes

- Enumerate every component of the platform (Go packages, web components, proto files, Helm templates, Docker images, extensions, CI, tests)
- Check each component for: correctness, consistency, dead code, missing tests, stale references
- Produce a findings report with categorized issues
- Generate follow-up `/opsx:propose` entries for each significant finding

## Capabilities

### New Capabilities
- `platform-audit`: Systematic audit of the entire UNCWORKS platform producing a findings report and follow-up proposals

### Modified Capabilities

(none)

## Impact

- No code changes in this change — output is findings + proposals
- Follow-up changes will address the findings
