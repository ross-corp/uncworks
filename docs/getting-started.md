# Getting started

## Prerequisites

- A local Kubernetes cluster. Docker Desktop, OrbStack, Rancher Desktop, k3d, and
  kind all work. OrbStack starts fastest on macOS.
- `kubectl` and `helm` on your PATH.
- The `uncworks` CLI. Download a binary from
  [GitHub Releases](https://github.com/ross-corp/uncworks/releases), or build it
  from a checkout with `./install.sh`, which needs Go 1.25 or later.

## Setup

```bash
uncworks setup
```

The wizard selects a kube context, checks that the cluster has enough resources,
asks for an LLM key and a GitHub token, and runs `helm upgrade --install`. The
resource floor is 2 CPU and 2 GiB. Allocate 4 CPU and 4 GiB for a usable
experience.

Pass every answer as a flag to skip the wizard.

```bash
uncworks setup \
  --context docker-desktop \
  --llm-key sk-... \
  --github-token ghp_... \
  --temporal-host temporal:7233
```

A local cluster can use the lighter preset. It serves NodePort 30300, requests
fewer resources, and disables Ollama.

```bash
uncworks setup --values deploy/helm/values.local.yaml
```

## Open the UI

```bash
uncworks open    # port-forward and open a browser
uncworks tui     # terminal UI
```

Docker Desktop, OrbStack, and Rancher Desktop expose NodePorts on `localhost`, so
http://localhost:30300 also works.

## Connect to a remote server

```bash
uncworks connect grpc.example.com:50055
uncworks tui    # now talks to the remote server
```

## First run

In the web UI, select New run, paste a repository URL, set a branch, write a
prompt, select a model tier, and select a mode. Leave the branch blank to use the
default branch. Select `single` for a one-shot run, or `spec-driven` for the
Plan, Execute, and Verify pipeline.

The CLI does the same thing.

```bash
uncworks runs create \
  --repo https://github.com/owner/repo \
  --prompt "Add a health check endpoint" \
  --model-tier default-cloud \
  --mode single
```

Every run goes through the `hybrid` approval gate by default. An LLM judge
reviews the diff, and then a human approves or rejects it in the UI. Change the
gate with `--approval-mode none|hitl|llm-judge|hybrid`.

## Status and teardown

```bash
uncworks status              # report pod health
uncworks teardown            # uninstall and keep the PVCs
uncworks teardown --purge    # uninstall and delete the PVCs
```

`--purge` deletes every workspace. The data cannot be recovered.

## Next

- [Creating runs](guides/creating-runs.md)
- [Spec-driven pipeline](guides/spec-driven.md)
- [Models](guides/models.md)
- [API reference](reference/api.md)
- [Custom resource reference](reference/crd.md)
