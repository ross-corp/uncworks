# Proposal: Remove Hardcoded Localhost Origins

## Problem

`cmd/apiserver/main.go` contains a hardcoded list of localhost origins (`localhost:3000`, `localhost:5173`, `127.0.0.1:3000`, `127.0.0.1:5173`) that are used as the default when `AOT_ALLOWED_ORIGINS` is not set. This makes the API server silently insecure in development and inflexible in production.

## Solution

Change the `parseAllowedOrigins` function:
1. If `AOT_ALLOWED_ORIGINS` is set, parse it as comma-separated origins (existing behavior)
2. If `AOT_ALLOWED_ORIGINS` is not set (empty), default to `*` for permissive dev mode
3. Remove the hardcoded localhost list entirely

This makes the behavior explicit: production deployments MUST set `AOT_ALLOWED_ORIGINS` via the Helm chart, and development mode is permissive by default.

## Scope

- `cmd/apiserver/main.go`: update `parseAllowedOrigins` default behavior
- Add unit test verifying origins are read from env
