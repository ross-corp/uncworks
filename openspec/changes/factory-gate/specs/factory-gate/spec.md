# factory-gate Specification

## ADDED Requirements

### Requirement: A change declares its tier, and the gate computes it from the diff

Every pull request that changes code MUST carry a tier. The gate MUST compute
the tier from the changed paths and from the change's spec, and MUST NOT read it
from the branch name, the pull request title, or a label. Those are writable by
the author the gate constrains, so reading a tier from one of them lets a change
lower its own bar.

The tiers are:

- `experiment`: a branch with no spec. It MAY target only another experiment
  branch, and it MUST NOT merge into the default branch.
- `T1`: a change confined to documentation, tests, or a single non-exported
  function. It needs no spec.
- `T2`: a change to one capability's implementation. It MUST have a merged spec.
- `T3`: a change that spans capabilities, alters a public surface, or touches
  deployment. It MUST have a merged spec and a review whose owner decision is
  recorded.

#### Scenario: A documentation-only change is T1 and needs no spec
- **POLARITY** positive
- **WHEN** the diff touches only `docs/` and `*.md`
- **THEN** the tier verdict is `T1` and passes with no change directory present

#### Scenario: A branch renamed to look small keeps its computed tier
- **POLARITY** negative
- **WHEN** a diff that alters a public surface is pushed to a branch named `docs/typo`
- **THEN** the tier verdict is `T3` and fails without a merged spec, because the
  branch name is not an input

#### Scenario: A multi-capability change without a merged spec fails
- **POLARITY** negative
- **WHEN** the diff touches two capabilities and `openspec/changes/<id>/` is
  absent from the base ref
- **THEN** the tier verdict fails and names the change id it looked for

### Requirement: The order check confirms the spec merged before the code

The gate MUST fail the order verdict when a pull request changes code for a
named change whose spec is not already present in the base ref. This is the
check that stops a spec from being written to match code that already exists.

The order verdict is advisory, because a small change legitimately carries its
spec and its code in one pull request.

#### Scenario: Code merged after its spec passes
- **POLARITY** positive
- **WHEN** `openspec/changes/<id>/proposal.md` exists in the base ref and the
  head ref changes only code
- **THEN** the order verdict passes

#### Scenario: Code that arrives before its spec is reported
- **POLARITY** negative
- **WHEN** the head ref changes code for change `<id>` and the base ref has no
  `openspec/changes/<id>/`
- **THEN** the order verdict fails and names the missing change

### Requirement: Spec conformance compares declared paths to changed paths

The gate MUST compare each changed path against the paths the change's
`## Impact` section declares. It MUST report a changed path that no declared
entry covers, and a declared entry that the diff never touches. Generated
directories are excluded, because nobody declares them and their churn would
drown the signal.

The verdict is advisory. A required check that is wrong often enough trains the
owner to override it, which costs more than the check returns.

#### Scenario: A diff confined to the declared paths conforms
- **POLARITY** positive
- **WHEN** every changed path is covered by an `## Impact` entry
- **THEN** the conformance verdict passes and lists no undeclared path

#### Scenario: An undeclared path is named
- **POLARITY** negative
- **WHEN** the diff changes `internal/server/grpc.go` and `## Impact` declares
  only `internal/gate/`
- **THEN** the conformance verdict reports `internal/server/grpc.go` as
  undeclared

### Requirement: The citations verdict runs the offline gate only

The gate MUST run the offline citation gate over the change's `citations.lock`,
and MUST NOT perform a live fetch. A live fetch would make the verdict a
function of network state, so an untouched pull request would start failing on
its own.

#### Scenario: A change with valid pins passes offline
- **POLARITY** positive
- **WHEN** every record's snapshot hashes to its recorded sha256 and every quote
  anchors
- **THEN** the citations verdict passes with no network request made

#### Scenario: An edited snapshot fails
- **POLARITY** negative
- **WHEN** a snapshot no longer hashes to the sha256 recorded for it
- **THEN** the citations verdict fails and names the claim id

### Requirement: An escalation is typed from the amendment diff

When a change's spec turns out to be wrong or infeasible, the author MUST raise
an escalation by opening an amendment to the spec. The gate MUST type that
escalation from what the amendment diff changes, and MUST NOT accept a type
declared by the person raising it.

A repository with one owner cannot delegate this judgment, so the typing is what
stops a scope change from being relabelled as a correction to skip a gate.

An amendment that changes a `## Behavior` criterion or a requirement's normative
text is an intent change. An amendment confined to wording, examples, or a
non-normative note is a correction.

While an amendment is open, the tier verdict MUST fail, so code cannot merge
against a spec that is under revision.

#### Scenario: An amendment to a Behavior criterion is an intent change
- **POLARITY** positive
- **WHEN** the amendment diff edits a line under `## Behavior`
- **THEN** the escalation is typed as an intent change and the tier verdict
  fails until the amendment merges or is withdrawn

#### Scenario: A raiser cannot declare the type
- **POLARITY** negative
- **WHEN** the author labels an amendment a correction and the diff edits a
  requirement's normative text
- **THEN** the gate types it as an intent change, and the label has no effect

### Requirement: Required checks are recomputed on the merge-group SHA

When the repository uses a merge queue, the gate MUST recompute every required
verdict on the merge-group SHA. A verdict computed on the pull-request SHA was
computed against a base the queue may have moved past, so it can be wrong for
the commit that actually lands.

#### Scenario: A queued pull request is rechecked
- **POLARITY** positive
- **WHEN** a pull request enters the merge queue
- **THEN** the workflow runs again on the merge-group SHA and the required
  verdicts are computed from that tree

#### Scenario: A stale pass does not carry over
- **POLARITY** negative
- **WHEN** the pull-request verdict passed but the merged tree fails a required
  verdict
- **THEN** the merge-group run fails and the pull request does not land

### Requirement: A soak record gates a flag retirement

When a spec declares a soak, the gate MUST open a soak record. The window MUST
start at the audit-logged targeting change, not at the merge. The gate MUST run
the query the spec names and compare the result against the declared threshold
and `min_samples`. A flag-retirement pull request MUST pass the soak verdict
only when the record is clean.

A soak whose window started at the merge would measure a period in which the
flag was off, which reports success for a change nobody ran.

#### Scenario: A clean soak record lets the flag retire
- **POLARITY** positive
- **WHEN** the named query is inside the threshold and the sample count is at or
  above `min_samples`
- **THEN** the soak verdict passes for the flag-retirement pull request

#### Scenario: Too few samples blocks retirement
- **POLARITY** negative
- **WHEN** the query is inside the threshold but the sample count is below
  `min_samples`
- **THEN** the soak verdict fails, because a threshold met by too few
  observations is not evidence
