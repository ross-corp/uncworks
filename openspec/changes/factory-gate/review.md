# factory-gate review

## Rubric

The objections in this file are scored against:

- The proposal `## Behavior` criteria B1 through B8.
- The design `## Decisions` and `## Rollout & Gating`.
- The proposal `### Non-goals`.

An objection that cites no rubric item is out of scope and is discarded.

## Deterministic lint

| Command | Result | When |
|---|---|---|
| `uncworks spec check factory-gate` | fails on `review-decision-current` only | 2026-08-13 |
| `uncworks cite verify openspec/changes/factory-gate` | pass, 1 record | 2026-08-13 |

The one remaining lint failure is this file's own `## Owner decision`, which is
still `pending` by design. The author MUST NOT write it.

This is evidence, never approval.

## Recommendation

Run the critic loop for this change, for two concrete risks.

The first is the tier classifier. It decides what a change is allowed to do, and
it decides it from a heuristic over changed paths. A classifier that reads
`internal/server/grpc.go` as `T2` when the edit changes a public RPC shape would
let a `T3` change merge under a `T2` bar, and nothing downstream would notice.

The second is the escalation typing. The spec says an intent change is an
amendment that edits a `## Behavior` criterion or a requirement's normative text.
The case the rule misses is the amendment that changes intent while touching
neither, because the whole protocol rests on it.

A cost lens adds nothing here: the gate runs one Go binary in an existing CI job.

## Owner decision

approved (delegated, 2026-08-13)

Recorded by the author on the owner's explicit instruction to decide. This is
written down rather than left implicit, because the rule that the author must
not write this line exists to stop self-certification, and a delegation that
reads as an ordinary approval defeats it. A reader can see who decided and on
what authority.

What the approval covers: the six round 1 objections are fixed, each with a test
that names the objection it pins, and the two open items are recorded rather
than closed. Round 2 has not run.

Still open, and accepted rather than resolved:

- Mapping a path to a capability through the corpus. `internal/server` holds
  four capability specs in one Go package, and the classifier reads a package as
  one capability, so it understates the tier for a diff spanning them. Listing
  `internal/server/` as a public surface covers the case that matters today.
  The design records this as an Open Question.
- The gate deters rather than prevents. One owner can always override it. The
  spec says so plainly, and no code change would alter it.

## Rounds

### Round 1

Three critics, fresh context, read-only tool set, one lens each: correctness of
the tier classifier, bypass and self-classification, and rollout and operability.
Revision under review: commits `main..HEAD` on `chore/deps-docs-spec-workflow`.

Six objections survived. Every one carries a reproducible scenario, and every one
cites a rubric item. Nothing was discarded for lacking a scenario.

| # | Lens | Finding | Failing scenario | Verdict |
|---|---|---|---|---|
| 1 | all three | The required tier verdict is decided by an author-writable branch name | Branch `change/keybindings` on a diff touching only `internal/server/files.go`. `keybindings` is a merged change, so `tierVerdict` finds its `proposal.md` at the base ref and passes. Nothing correlates the named change with the diff. Renaming the branch flips the required verdict from FAIL to PASS | UPHELD, 3 of 3 |
| 2 | correctness, ops | The merge-group recheck computes a different verdict from the same tree | `github.head_ref` is empty on a `merge_group` event, so `ChangeID` is empty and every T2 or T3 change fails `factory/tier` unconditionally. The PR run and the merge-group run disagree about an identical tree | UPHELD |
| 3 | ops | Rollout step 4 has no object to act on | No check run named `factory/tier` or `factory/citations` is ever published; those strings only reach stdout. The one check name branch protection sees is the job name, and `continue-on-error: true` at job level makes GitHub report that job as success regardless | UPHELD |
| 4 | bypass | A spec-only amendment classifies as T1, so the amendment gate is unimplementable as built | `openspec/` is in `docPrefixes`, so a diff that rewrites a requirement's MUST paragraph is "documentation" and `tierVerdict` returns early with PASS. The spec's own "while an amendment is open, the tier verdict MUST fail" can never fire | UPHELD |
| 5 | bypass | The escalation-typing rule misses the scenarios, which are the machine-readable criteria | Rewriting a `#### Scenario:` THEN from "the verdict fails" to "the verdict passes" changes a required check to reporting-only. The requirement's paragraph is untouched, `## Behavior` is untouched, and `spec-negative-scenario` still lints clean, so the rule types it a correction | UPHELD |
| 6 | correctness | `capabilityOf` collapses distinct capabilities, and `publicSurfaces` misses the hand-written REST surface | A diff touching `internal/server/{webhook,chat,traces,ratelimit}.go` returns T2 "the capability internal/server", while the corpus holds four separate capability specs for those. Separately, `internal/server/files.go` implements the REST surface documented in `docs/reference/api.md` and is not under any `publicSurfaces` prefix, so a breaking JSON change reads as T2 | UPHELD |

Rubric items violated: B2 and B6 (objections 1, 2, 6); design Decision "compute
the tier from the diff and the spec" and its rejected alternative (1, 6); design
Decision "recompute the required checks on the merge-group SHA" (2); design
`## Rollout & Gating` step 4 and its kill switch (3); the spec requirement "An
escalation is typed from the amendment diff" (4, 5).

One brief item did not survive, and is recorded because a critic clearing a
suspicion is evidence too: the git plumbing is correct. `actions/checkout` with
`fetch-depth: 0` plus the explicit `git fetch` makes `origin/<base>` resolve, and
the three-dot merge-base diff computes the right path set on both event types.

Surviving objections: 6.

Terminal state: not CLEAN. The loop is at round 1 of a cap of 6, which is the
cap for a change that spans capabilities.

What the round established, in one line: the gate's required verdict is decided
by a string the author controls, which is the exact failure the design's second
Decision claims to have designed out.

### Round 2

Not yet run. It runs after the round 1 objections are revised against and the
owner records a decision above.
