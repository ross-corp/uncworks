# Proposal: Rename User-Facing "AOT" to "UNCWORKS"

## Problem

The project has been rebranded to UNCWORKS, but many user-facing strings still reference the old name "AOT". This creates confusion for users encountering mixed branding in the web UI, Helm chart output, CLI messages, Taskfile output, and documentation.

## Solution

Rename all user-facing "AOT" references to "UNCWORKS" across:
- Web UI title (`web/index.html`)
- Helm chart description and NOTES.txt output
- Taskfile echo/banner strings
- Documentation markdown files
- CLI log messages in `cmd/apiserver/main.go` and `cmd/aot/main.go`

## Scope

- User-facing text only (titles, descriptions, banners, docs)
- Do NOT rename Go package names (keep `aot` internally)
- Do NOT rename CRD group (`aot.uncworks.io`)
- Do NOT rename environment variable prefixes (`AOT_*`)
- Do NOT rename Helm chart name or template helpers
