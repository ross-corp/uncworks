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

- Decision: the tier verdict requires the named change's Impact to claim the
  diff, not merely to exist.
- Alternative rejected: checking only that `openspec/changes/<id>/proposal.md`
  is present at the base ref. The round 1 review found, from three critics
  independently, that the change id reaches the gate from a branch name the
  author writes, so any diff could borrow any merged change's spec and pass the
  required verdict. Checking existence alone reintroduces the bar-lowering the
  second Decision rejects, one layer down.

- Decision: read the change id from a `Change-Id:` commit trailer as well as
  from the branch.
- Alternative rejected: the branch name alone. A merge queue rewrites the
  branch, so `github.head_ref` is empty on a `merge_group` event and every T2
  and T3 change would fail the recheck for want of a change id. The trailer
  travels with the commit, so the merge-group run resolves the same change the
  pull-request run did, which is what makes the recheck a recheck.

- Decision: publish one job per required verdict, named `factory/tier` and
  `factory/citations`, and carry no `continue-on-error` on either.
- Alternative rejected: one job with `continue-on-error: true`. The check names
  the design asks the owner to mark required were never published, so branch
  protection saw only the job's own name, and `continue-on-error` made GitHub
  report that job as success whatever the step returned. An unrequired check
  that fails does not block a merge, so the staged rollout works without
  pretending the job passed, and the kill switch stays a settings change.

- Decision: treat a diff that touches `openspec/specs/` as its own tier, which
  never passes on its own.
- Alternative rejected: counting `openspec/` as documentation. That made a spec
  amendment T1, so the one diff shape the escalation protocol exists for was the
  one shape the classifier could not fail.

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
3. Add three CI jobs: `factory/tier`, `factory/citations`, and an advisory job
   for the other two verdicts. Gate: each runs on a pull request and its output
   is readable.
4. Apply: mark `factory/tier` and `factory/citations` required in branch
   protection. Gate: both have run green on at least one real pull request.
   This is an owner gate, because it changes what can merge.

The two required jobs carry no `continue-on-error`. An unrequired check that
fails does not block a merge, so a failing job before step 4 reports honestly
rather than reporting success. Only the advisory job absorbs its exit code.

Kill switch: the required checks are branch-protection settings, so unmarking
them restores the previous behavior with no code change. Before step 4 nothing
blocks, because nothing is required.

## Risks / Trade-offs

- A tier classifier that reads the diff will misclassify some change, mitigated
  by making the classification visible in the job output and by keeping the
  owner's judgment as the override. The classifier reports its reason, so a
  wrong verdict is arguable rather than opaque.
- The classifier groups a Go package as one capability, while the corpus can
  hold several capability specs implemented in one package. `internal/server`
  holds four. The round 1 review found this understates the tier for a diff
  spanning them. It is mitigated for the public-surface half by listing
  `internal/server/` as a public surface, so any change there is T3. The general
  case, mapping a path to a capability through the corpus, is open and recorded
  in Open Questions.
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
- How a path maps to a capability in the corpus. The classifier currently uses
  the package directory, which is right for most of the tree and wrong for
  `internal/server`, where four capability specs share one package. Reading the
  mapping from `openspec/specs/` needs the specs to record which paths implement
  them, which none of them do today.
