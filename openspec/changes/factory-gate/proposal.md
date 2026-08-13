# factory-gate

## Why

Nothing today connects a request to a spec, or a spec to the code that claims to
implement it. A change can merge with no spec, a spec can merge and never be
implemented, and a spec that turns out to be wrong is silently rewritten by
whoever hits it first. The repository has 71 specs under `openspec/specs/` and no
check that any pull request respects one.

This repository has one owner, so every gate here is a commitment device against
that owner rather than a handoff between people. The value is the same: the gate
makes the moment of scope change visible instead of letting it happen inside an
editor.

## What Changes

- Add `uncworks gate check`, which computes four verdicts for a pull request:
  tier, spec-conformance, order, and citations.
- Add a tier classifier that reads the diff and the spec, so a change declares
  its blast radius rather than having it inferred at review time.
- Add a spec-conformance check that compares the changed paths against the paths
  the change's `## Impact` declares.
- Add an order check that confirms the spec merged before the code that claims
  to implement it.
- Add a CI job that runs `uncworks gate check` on every pull request and on the
  merge group. It runs natively rather than through Dagger, because every
  verdict needs real git refs and the Dagger source directory carries no usable
  history.
- Record the escalation protocol in `openspec/specs/factory-gate/spec.md`: an
  escalation is typed from the amendment diff, not from the person raising it.

### Non-goals

- The webhook service, the HMAC verification, and the check-run API writes. The
  CI job covers the same ground without a service to operate, and a service can
  replace it later without changing a verdict.
- Soak records and flag retirement. The soak window starts at an audit-logged
  targeting change, and this repository has no feature-flag service to read that
  from. The spec records the contract so the check can be added without a
  redesign.
- Slack and Linear intake. Intake is a GitHub issue or a local invocation.
- The `experiment` mode branch policy. Nothing enforces branch targeting yet.

## Behavior

- B1: `uncworks gate check --change <id> --base <ref> --head <ref>` exits 0 when
  every required verdict passes, and 1 when any required verdict fails.
- B2: The tier verdict is required. A change whose tier demands a merged spec
  fails when `openspec/changes/<id>/` is absent from the base ref.
- B3: The spec-conformance verdict is advisory. It names each changed path that
  no `## Impact` entry covers, and each declared path the diff never touches.
- B4: The order verdict is advisory. It fails when the head ref changes code and
  the spec for the named change is not present in the base ref.
- B5: The citations verdict is required. It runs the offline gate, so it reaches
  no network and returns the same verdict for the same tree.
- B6: A tier is computed from the diff and the spec, never from the pull request
  title or the branch name, so renaming a branch cannot lower the bar.
- B7: `uncworks gate check --json` emits one object per verdict, so a CI job
  can render it without parsing prose.
- B8: `task test:go` passes with the gate package present.

## Impact

- Code: `internal/gate/`, `cmd/uncworks/gate.go`
- Specs: `openspec/specs/factory-gate/spec.md`
- CI: `.github/workflows/ci.yml`
- Impactful actions, each of which becomes an owner gate in tasks.md:
  making `factory/tier` and `factory/citations` required checks on the default
  branch, because that changes what can merge.
- Gating signal: the CI job runs advisory-only until the owner marks the two
  required checks required in the branch protection settings.
- Judgment that stays with the owner: whether a tier verdict is right for a
  given change, and whether an escalation is a genuine intent change or a
  developer routing around a gate.
- External-factual claims: one, pinned in `citations.lock`. GitHub's
  `merge_group` event is what lets a required check be recomputed on the
  merge-group SHA rather than on the pull-request SHA
  [cite: github-merge-group-event].
