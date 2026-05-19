# Getting started

## Prerequisites

- A local Kubernetes cluster. Docker Desktop, OrbStack (fastest on macOS), Rancher Desktop, k3d, or kind all work.
- `kubectl` and `helm` on PATH.
- The `uncworks` CLI: `brew install uncworks/tap/uncworks`, or grab a binary from [GitHub Releases](https://github.com/uncworks/uncworks/releases).

## Setup

```bash
uncworks setup
```

The wizard picks a kube context, checks resources (2 CPU / 2 GiB floor; 4/4 recommended), asks for an LLM key + GitHub token, and runs `helm upgrade --install`. For non-interactive use:

```bash
uncworks setup \
  --context docker-desktop \
  --llm-key sk-... \
  --github-token ghp_... \
  --temporal-host temporal:7233
```

Local clusters can use the bundled lighter preset (NodePort 30300, reduced requests, Ollama off):

```bash
uncworks setup --values deploy/helm/values.local.yaml
```

## Open it

```bash
uncworks open    # port-forward + browser
uncworks tui     # terminal UI
```

Docker Desktop / OrbStack / Rancher expose NodePorts on `localhost`, so http://localhost:30300 also works.

## Remote server

```bash
uncworks connect grpc.example.com:50055
uncworks tui    # now talks to the remote server
```

## First run

In the web UI: "New run", paste a repo URL, set a branch (default if blank), write a prompt, pick a model tier, pick a mode (`single` for a one-shot, `spec-driven` for Plan/Execute/Verify), submit.

Or from the CLI:

```bash
uncworks runs create \
  --repo https://github.com/owner/repo \
  --prompt "Add a health check endpoint" \
  --model-tier default-cloud \
  --mode single
```

By default every run goes through the **hybrid** approval gate: an LLM judge reviews the diff, and then a human approves or rejects in the UI. Override with `--approval-mode none|hitl|llm-judge|hybrid`.

## Status, teardown

```bash
uncworks status              # pod health
uncworks teardown            # uninstall, keep PVCs
uncworks teardown --purge    # uninstall + delete PVCs (workspace data is gone)
```

## Next

- [Creating runs](guides/creating-runs.md)
- [Spec-driven pipeline](guides/spec-driven.md)
- [Models](guides/models.md)
- [API reference](reference/api.md)
- [CRD reference](reference/crd.md)
