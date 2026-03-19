# New Run View

Triggered by `n` from the run list. Two modes: quick prompt or full spec.

## Quick Prompt Mode (default)

```
┌─────────────────────────────────────────────────────────────────────┐
│  AOT                                                    ⌘K  ? help │
├─────────────────────────────────────────────────────────────────────┤
│  New Run                                                esc cancel │
│                                                                     │
│  Repo    roshbhatia/neph.nvim  main  ▾                             │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ fix the auth middleware to validate JWT tokens properly        ││
│  │                                                                ││
│  │                                                                ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  Model   qwen3:8b (local) ▾     TTL  15m ▾     Mode  single ▾     │
│                                                                     │
│                                              [Refine with AI]  Run │
└─────────────────────────────────────────────────────────────────────┘
```

## After "Refine with AI" (chat mode)

```
┌─────────────────────────────────────────────────────────────────────┐
│  AOT                                                    ⌘K  ? help │
├─────────────────────────────────────────────────────────────────────┤
│  New Run                                                esc cancel │
│                                                                     │
│  Repo    roshbhatia/neph.nvim  main  ▾                             │
│                                                                     │
│  ┌ Conversation ────────────────────────────────────────────────── ┐│
│  │                                                                ││
│  │  you   fix the auth middleware to validate JWT tokens          ││
│  │                                                                ││
│  │  aot   I'll create a spec-driven run with these criteria:      ││
│  │        - Add JWT validation to authMiddleware()                ││
│  │        - Check Authorization header for Bearer token           ││
│  │        - Verify against JWT_SECRET env var                     ││
│  │        - Return 401 for missing/invalid tokens                 ││
│  │        - Add tests for valid, expired, and missing tokens      ││
│  │                                                                ││
│  │        Model: qwen3:8b (local)                                 ││
│  │        Mode: spec-driven (plan → execute → verify)             ││
│  │                                                                ││
│  │        Shall I proceed?                                        ││
│  │                                                                ││
│  │  you   also add rate limiting, 100 requests per minute         ││
│  │                                                                ││
│  │  aot   Updated. Adding rate limiting middleware:               ││
│  │        - 100 req/min per IP on /api/users                      ││
│  │        - Return 429 with Retry-After header                    ││
│  │                                                                ││
│  └────────────────────────────────────────────────────────────────┘│
│  ┌────────────────────────────────────────────────────────────────┐│
│  │ > type to refine...                                    send ↵ ││
│  └────────────────────────────────────────────────────────────────┘│
│                                                                     │
│                                                         Run Agent  │
└─────────────────────────────────────────────────────────────────────┘
```

## Full Spec Mode (toggle via tab or ⌘+S)

```
┌─────────────────────────────────────────────────────────────────────┐
│  AOT                                                    ⌘K  ? help │
├─────────────────────────────────────────────────────────────────────┤
│  New Run                                                esc cancel │
│                                                                     │
│  Repo    roshbhatia/neph.nvim  main  ▾                             │
│                                                                     │
│  [Prompt]  [Spec]                                                   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ ## Requirements                                                ││
│  │                                                                ││
│  │ ### Add JWT auth to /api/users                                 ││
│  │                                                                ││
│  │ #### Scenario: unauthorized request                            ││
│  │ - **WHEN** request has no Authorization header                 ││
│  │ - **THEN** return 401 with error message                       ││
│  │                                                                ││
│  │ #### Scenario: valid token                                     ││
│  │ - **WHEN** request has valid Bearer token                      ││
│  │ - **THEN** proceed to route handler                            ││
│  │                                                                ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  Model   qwen3:8b (local) ▾     TTL  15m ▾     Mode  spec-driven  │
│                                                                     │
│                                              [Refine with AI]  Run │
└─────────────────────────────────────────────────────────────────────┘
```

## Behavior

- **Prompt** is the default input — a text box, like a chat input
- **"Refine with AI"** opens the chat refinement panel (calls the API to plan)
- **"Run"** launches immediately with the prompt/spec as-is
- **Spec tab** swaps the text input for a Monaco editor with markdown highlighting
- **Mode** auto-switches to `spec-driven` when Spec tab is active
- Repo selector shows recently used repos, with autocomplete
- Model/TTL/Mode are collapsed by default, shown as a single line of defaults

## Collapsed Config (default state)

```
│  qwen3:8b · 15m · single                              [▾ settings] │
```

Clicking "settings" expands to show Model, TTL, Mode, and pipeline config.

## Comments

<!-- @unc: -->
<!-- @claude: -->
