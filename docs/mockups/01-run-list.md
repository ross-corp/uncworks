# Run List View

The default view. k9s-style resource browser for agent runs.

## Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  AOT                                                    ⌘K  ? help │
├─────────────────────────────────────────────────────────────────────┤
│  Runs (18)                                    / filter  n new run  │
├────┬──────────────────────┬────────┬──────────┬────────┬───────────┤
│    │ NAME                 │ STATUS │ STAGE    │ MODEL  │ AGE       │
├────┼──────────────────────┼────────┼──────────┼────────┼───────────┤
│  ▸ │ fix-auth-bug         │ ●  run │ execute  │ qwen3  │ 2m        │
│    │ add-rate-limiting    │ ●  run │ verify   │ qwen3  │ 5m        │
│    │ update-readme        │ ✓  ok  │          │ qwen3  │ 12m       │
│    │ refactor-handlers    │ ✗ fail │          │ llama  │ 1h        │
│    │ fix-login-page       │ ✓  ok  │          │ qwen3  │ 3h        │
│    │ deploy-staging       │ ✗ fail │          │ qwen3  │ 5h        │
│    │ add-user-search      │ ✓  ok  │          │ qwen3  │ 1d        │
│    │ migrate-to-prisma    │ ●  run │ plan     │ cloud  │ 2m        │
│    │                      │        │          │        │           │
│    │                      │        │          │        │           │
│    │                      │        │          │        │           │
├────┴──────────────────────┴────────┴──────────┴────────┴───────────┤
│  j/k navigate  enter detail  n new  / filter  d delete  c clone    │
└─────────────────────────────────────────────────────────────────────┘
```

## Behavior

- **j/k** or **↑/↓** — move selection
- **enter** — open run detail view
- **n** — open new run input
- **/** — focus filter input (filters NAME, STATUS, MODEL)
- **d** — delete selected run (with confirmation)
- **c** — clone selected run (opens new run with pre-filled data)
- **⌘K** — command palette
- **1/2/3/4** — quick filter: all / active / succeeded / failed
- **?** — show help overlay with all shortcuts

## Status Indicators

```
●  running (green dot, pulsing)
✓  succeeded (green check)
✗  failed (red x)
◎  pending (gray circle)
⏸  waiting for input (yellow pause)
⊘  cancelled (gray slash)
```

## Stage Column

Only shown for spec-driven runs. Empty for single-mode.

```
plan     — generating spec
execute  — implementing
verify   — evaluating against spec
```

## Filter Bar

When `/` is pressed, a filter input appears inline:

```
┌─────────────────────────────────────────────────────────────────────┐
│  Runs (18)                                                          │
│  / failed█                                                          │
├────┬──────────────────────┬────────┬──────────┬────────┬───────────┤
│    │ refactor-handlers    │ ✗ fail │          │ llama  │ 1h        │
│    │ deploy-staging       │ ✗ fail │          │ qwen3  │ 5h        │
└────┴──────────────────────┴────────┴──────────┴────────┴───────────┘
```

Esc clears the filter and returns to the full list.

## Comments

<!-- @unc: -->
<!-- @claude: -->
