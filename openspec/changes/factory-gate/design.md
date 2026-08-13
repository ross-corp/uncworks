# factory-gate design

> The keywords MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY in this document are
> to be interpreted as described in [RFC 2119](https://datatracker.ietf.org/doc/html/rfc2119).

## Context

The repository already holds every piece this needs except the wiring.

`openspec/specs/` holds 71 capability specs and is the durable corpus.
`openspec/changes/<id>/` holds an in-flight change. `internal/specutil` parses
both and rubric-lints a change. `internal/citelock` gates the pinned citations
offline. `internal/temporal` already runs a Plan, Execute, Verify pipeline
against a change, and `internal/github` already speaks to the GitHub API.

What is missing is the join: nothing relates a pull request to the change it
claims to implement, and nothing compares the diff against what the change said
it would touch.

This design extends `internal/specutil`, which already parses the artifacts,
rather than adding a second parser beside it.

## Goals / Non-Goals

Goals:

- One command computes every verdict, so CI and a local run agree by
  construction.
- Every verdict is a pure function of two git refs and the files in them.
- A verdict names the specific path or artifact that failed, so the fix is
  obvious without opening the tool.

Non-goals:

- A long-lived service. See the proposal's Non-goals for why.
- Deciding whether a spec is a good spec. That is the adversarial review's job,
  and it is not deterministic.

## Decisions

- Decision: run the gate as a CI job invoking `uncworks gate check`, not as a
  webhook service that writes check runs.
- Alternative rejected: the webhook service from the original design. It needs
  an HMAC secret, a public endpoint, a persistent store for soak records, and an
  operator. Every verdict it would compute is a pure function of two refs, so a
  job in the workflow computes the same thing with nothing to run. The service
  becomes worth it when soak records need durable state, and it can replace the
  job then without changing a verdict.

- Decision: compute the tier from the diff and the spec.
- Alternative rejected: reading the tier from a label or the branch name. Both
  are writable by the same person the gate is meant to slow down, so a change
  could lower its own bar by renaming a branch. Deriving it from the diff means
  the only way to reduce the tier is to reduce the change.

- Decision: make tier and citations required, and make spec-conformance and
  order advisory.
- Alternative rejected: making all four required. Spec-conformance compares
  declared paths to changed paths, and a legitimate change routinely touches a
  file the author did not think to declare, such as a generated file or a test
  helper. A required check that is wrong often enough trains the owner to
  override it, which costs more than the check returns. Order fails on a
  legitimate combined spec-and-code pull request, which is the right shape for a
  small change.

- Decision: type an escalation from the amendment diff, not from the person who
  raises it.
- Alternative rejected: letting the raiser declare the escalation type. With one
  owner that is self-classification, and the failure mode is exactly the one the
  gate exists to catch: relabelling a scope change as a bugfix to skip a gate.
  Reading the type from what the amendment actually changes removes the choice.

- Decision: keep `openspec/specs/` as the corpus that spec-conformance reads.
- Alternative rejected: sysinit's model, where the acceptance criteria live in
  the proposal and there is no corpus. The Verify stage in `internal/temporal`
  reads the corpus today, and the conformance check needs a stable place to ask
  "which capability does this path belong to".

- Decision: recompute the required checks on the merge-group SHA.
- Alternative rejected: trusting the pull-request SHA. The pull-request result is
  computed against a base the merge queue may have moved past, so a check that
  passed on the pull request can be wrong for the commit that actually lands.
  GitHub emits a separate event for this, which runs a workflow when a pull
  request is added to a merge queue [cite: github-merge-group-event].

## Rollout & Gating

1. Land `internal/gate` and `cmd/uncworks/gate.go`. Gate: `task test:go` exits 0.
2. Land the spec under `openspec/specs/factory-gate/spec.md`. Gate:
   `uncworks spec check factory-gate` exits 0.
3. Add the CI job, reporting only. Gate: the job runs on a pull request and its
   output is readable.
4. Apply: mark `factory/tier` and `factory/citations` required in branch
   protection. Gate: the job has run green on at least one real pull request.
   This is an owner gate, because it changes what can merge.

Kill switch: the required checks are branch-protection settings, so unmarking
them restores the previous behavior with no code change. Until step 4, every
verdict is advisory by construction.

## Risks / Trade-offs

- A tier classifier that reads the diff will misclassify some change, mitigated
  by making the classification visible in the job output and by keeping the
  owner's judgment as the override. The classifier reports its reason, so a
  wrong verdict is arguable rather than opaque.
- Spec-conformance will produce false positives on generated files, mitigated by
  keeping it advisory and by excluding `gen/` from the comparison.
- A gate with one owner can always be overridden by that owner, so it deters
  rather than prevents. That is the honest limit and it MUST be stated: this
  makes scope change visible, it does not make it impossible.
- The four in-flight changes predate the schema and fail `uncworks spec check`,
  so the tier check MUST NOT require a clean rubric for a change that already
  exists. It requires the spec to be present, not to be perfect.

## Migration Plan

No data migrates. The gate reads git refs and files.

Rollback is removing the CI job, or unmarking the required checks. Before either
step 3 or step 4, run `uncworks gate check` locally against a recent merged pull
request and confirm the verdicts match what a reader would expect. After step 4,
confirm that a deliberately non-conforming branch is blocked.

## Open Questions

- Where the soak window's start timestamp comes from. The design assumes an
  audit-logged targeting change, and this repository has no feature-flag service
  yet. The spec records the contract so the answer can be filled in later.
- Whether `experiment` mode needs branch-targeting enforcement, or whether the
  tier check naming the mode is enough.
