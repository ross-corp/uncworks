# Run Detail View

Opened by pressing `enter` on a run in the list, or navigating to `:run ar-ju91iv`.

## Live Activity Feed (default tab)

```
┌─────────────────────────────────────────────────────────────────────┐
│  AOT                                                    ⌘K  ? help │
├─────────────────────────────────────────────────────────────────────┤
│  ar-ju91iv · fix-auth-bug                      ●  running  execute │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━░░░░░░░░░░  Plan ✓  Execute...  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  10:32:01  system   Agent started                                   │
│                                                                     │
│  10:32:01  user     fix the auth middleware to validate JWT tokens  │
│                                                                     │
│  10:32:03  agent    I'll start by examining the current auth        │
│                     middleware to understand the structure.          │
│                                                                     │
│  10:32:04  tool     bash                                            │
│            ┌──────────────────────────────────────────────────┐     │
│            │ ls src/middleware/                                │     │
│            └──────────────────────────────────────────────────┘     │
│            ┌ result ──────────────────────────────────────────┐     │
│            │ auth.ts  cors.ts  logger.ts  index.ts            │     │
│            └──────────────────────────────────────────────────┘     │
│                                                                     │
│  10:32:06  tool     read                                            │
│            ┌──────────────────────────────────────────────────┐     │
│            │ src/middleware/auth.ts                            │     │
│            └──────────────────────────────────────────────────┘     │
│            ┌ result (245 chars) ──────────────── ▸ expand ───┐     │
│            │ export function authMiddleware(req, res, next)   │     │
│            │ { /* current: just checks if header exists */ }  │     │
│            └──────────────────────────────────────────────────┘     │
│                                                                     │
│  10:32:08  agent    The current middleware only checks for header   │
│                     existence. I'll add proper JWT verification     │
│                     using jsonwebtoken.                             │
│                                                                     │
│  10:32:09  tool     write                                           │
│            ┌ diff ────────────────────────────────────────────┐     │
│            │ src/middleware/auth.ts                            │     │
│            │ -  if (!req.headers.authorization) {              │     │
│            │ +  const token = req.headers.authorization?.      │     │
│            │ +    replace('Bearer ', '');                      │     │
│            │ +  if (!token) {                                  │     │
│            │ +    return res.status(401).json({                │     │
│            │ +      error: 'Missing token' });                 │     │
│            │ +  }                                              │     │
│            │ +  try {                                          │     │
│            │ +    jwt.verify(token, process.env.JWT_SECRET);   │     │
│            │ +  } catch {                                      │     │
│            │ +    return res.status(401).json({                │     │
│            │ +      error: 'Invalid token' });                 │     │
│            │ +  }                                              │     │
│            └──────────────────────────────────────────────────┘     │
│                                                                     │
│  10:32:10  agent    Added JWT verification. Now adding tests...     │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  1 activity  2 files  3 shell  4 traces  5 verify    esc back      │
└─────────────────────────────────────────────────────────────────────┘
```

## Stage Progress Bar

For spec-driven runs, shows pipeline progress:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━░░░░░░░░░░  Plan ✓  Execute...  Verify

Legend:
  ✓  completed stage
  ... in-progress stage
  ○  pending stage
  ✗  failed stage (red)
```

For single-mode runs, no progress bar — just the status badge.

## Tab Views (number keys)

### 1 — Activity (default, shown above)

Live agent conversation feed with tool calls, results, diffs, and agent text.

### 2 — Files

```
├─────────────────────────────────────────────────────────────────────┤
│  Files  /workspace                                      esc back   │
│                                                                     │
│  ▸ .aot/                                                            │
│  ▸ src/                                                             │
│    ├── middleware/                                                   │
│    │   ├── auth.ts              1.2 KB  modified 2m ago             │
│    │   ├── auth.test.ts         0.8 KB  new                        │
│    │   ├── cors.ts              0.3 KB                              │
│    │   └── index.ts             0.2 KB                              │
│    └── routes/                                                      │
│        └── users.ts             0.5 KB                              │
│    package.json                 0.4 KB                              │
│    tsconfig.json                0.3 KB                              │
│                                                                     │
│  enter open  q close preview  / search                              │
├─────────────────────────────────────────────────────────────────────┤
```

### 3 — Shell

```
├─────────────────────────────────────────────────────────────────────┤
│  Shell  ar-ju91iv                                       esc back   │
│                                                                     │
│  root@agentrun-ar-ju91iv:/workspace$ ls                             │
│  .aot  src  package.json  tsconfig.json                             │
│  root@agentrun-ar-ju91iv:/workspace$ npm test                       │
│                                                                     │
│  > auth-api@1.0.0 test                                              │
│  > jest                                                             │
│                                                                     │
│   PASS  src/middleware/auth.test.ts                                  │
│    ✓ returns 401 for missing token (3ms)                            │
│    ✓ returns 401 for invalid token (2ms)                            │
│    ✓ passes valid token through (1ms)                               │
│                                                                     │
│  Tests: 3 passed, 3 total                                           │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
```

### 4 — Traces

```
├─────────────────────────────────────────────────────────────────────┤
│  Traces  ar-ju91iv                                      esc back   │
│                                                                     │
│  10:32:01 ├── agent_started                             0ms        │
│  10:32:03 ├── llm_call (qwen3:8b)                      1.2s       │
│  10:32:04 ├── tool: bash (ls src/middleware/)            0.1s       │
│  10:32:05 ├── llm_call (qwen3:8b)                      0.8s       │
│  10:32:06 ├── tool: read (src/middleware/auth.ts)        0.05s      │
│  10:32:07 ├── llm_call (qwen3:8b)                      1.5s       │
│  10:32:08 ├── tool: write (src/middleware/auth.ts)       0.02s      │
│  10:32:09 ├── tool: write (src/middleware/auth.test.ts)  0.02s      │
│  10:32:10 ├── llm_call (qwen3:8b)                      0.9s       │
│  10:32:10 └── agent_ended                               0ms        │
│                                                                     │
│  Total: 9.1s  LLM: 4.4s (48%)  Tools: 0.2s  Tokens: 2,847         │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
```

### 5 — Verify (spec-driven only)

```
├─────────────────────────────────────────────────────────────────────┤
│  Verification  ar-ju91iv                                esc back   │
│                                                                     │
│  PASSED  in 3.2s                                                    │
│                                                                     │
│  ✓  Task completion      23/23 tasks                               │
│  ✓  Spec validation      valid                                     │
│  ✓  auth.ts exists       src/middleware/auth.ts                     │
│  ✓  auth.test.ts exists  src/middleware/auth.test.ts               │
│  ✓  npm test             exit 0 (3 passed)                         │
│  ▸  LLM judge            qwen3:8b                                  │
│     ├── ✓  unauthorized request returns 401                        │
│     ├── ✓  valid token passes through                              │
│     └── ✓  JWT_SECRET env var used for verification                │
│                                                                     │
│  Attempt 1 of 3                                                     │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
```

## HITL (Human-in-the-Loop) Overlay

When the agent asks for input (`waiting_for_input` phase):

```
├─────────────────────────────────────────────────────────────────────┤
│  10:32:15  agent    I found two auth approaches in the codebase.   │
│                     Which should I use?                             │
│                     1. JWT with jsonwebtoken                        │
│                     2. Session-based with express-session           │
│                                                                     │
│  ┌ Agent is waiting for input ─────────────────────────────────── ┐│
│  │ > use JWT, we don't want server-side sessions             send ││
│  └────────────────────────────────────────────────────────────────┘│
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
```

## Header Detail (collapsed by default)

Press `i` to toggle info overlay:

```
┌─────────────────────────────────────────────────────────────────────┐
│  ID          ar-ju91iv                                              │
│  Created     2026-03-18 10:31:55                                    │
│  Duration    2m 15s                                                 │
│  Repo        roshbhatia/neph.nvim (main)                           │
│  Model       qwen3:8b (local)                                      │
│  Mode        spec-driven                                            │
│  Stage       execute (attempt 1/3)                                  │
│  Pod         agentrun-ar-ju91iv-7c75-sglz7                         │
│  Prompt      fix the auth middleware to validate JWT tokens...      │
├─────────────────────────────────────────────────────────────────────┤
```

## Comments

<!-- @unc: -->
<!-- @claude: -->
