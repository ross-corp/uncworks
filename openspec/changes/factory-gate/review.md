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
A critic with a correctness lens should try to construct that diff.

The second is the escalation typing. The spec says an intent change is an
amendment that edits a `## Behavior` criterion or a requirement's normative text.
A critic with an adversarial lens should try to find the amendment that changes
intent while touching neither, because that is the case the rule misses and the
whole protocol rests on it.

A cost lens adds nothing here: the gate runs one Go binary in an existing CI job.

## Owner decision

pending

The owner writes this line, not the author. Replace it with `approved`,
`not-run`, or `halted at round <n>`. Silence is not consent, and this change is
not done while the line reads `pending`.

## Rounds

### Round 1

Not yet run. It runs after the owner records a decision above.

| Lens | Finding | Failing scenario | Verdict |
|---|---|---|---|
| | | | |

Surviving objections: not yet measured

Terminal state: not yet reached
