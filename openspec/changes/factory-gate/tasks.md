# factory-gate tasks

## 1. The gate package

- **SHAPE** graph
- **MERGE** 1.5

- [x] 1.1 Add the tier classifier in `internal/gate`, reading the changed paths
      and the change's spec. It follows the pure-function shape of
      `internal/specutil` `writes:` internal/gate/tier.go
- [x] 1.2 Add the spec-conformance verdict, comparing changed paths against the
      proposal `## Impact` entries `deps:` 1.1 `writes:` internal/gate/conformance.go
- [x] 1.3 Add the order verdict, which reads the base ref for the change
      directory `deps:` 1.1 `writes:` internal/gate/order.go
- [x] 1.4 Add the citations verdict, delegating to `internal/citelock`
      `deps:` 1.1 `writes:` internal/gate/citations.go
- [x] 1.5 Add `uncworks gate check`, which runs every verdict and reports them as
      text or JSON `deps:` 1.2, 1.3, 1.4 `writes:` internal/gate/gate.go, cmd/uncworks/gate.go
- [x] 1.6 Adversarial review of this phase against the proposal Behavior
      criteria. Run `uncworks spec check factory-gate` and append a round block
      to review.md `deps:` 1.5
- [ ] 1.7 Revise against the round 1 objections that remain open, then run
      round 2 `deps:` 1.6

## 2. CI wiring

- **SHAPE** graph
- **MERGE** 2.3

- [x] 2.1 Run the gate natively rather than through Dagger. Every verdict needs
      real git refs, and the Dagger source directory carries no usable history
      `writes:` .github/workflows/ci.yml
- [x] 2.2 Add the workflow job on `pull_request` and `merge_group`, with
      continue-on-error until the owner marks the required checks required
      `deps:` 2.1 `writes:` .github/workflows/ci.yml
- [x] 2.3 Adversarial review of this phase. Run `uncworks spec check factory-gate`
      and append a round block to review.md `deps:` 2.2

## 3. Rollout

- [ ] 3.1 Apply: mark `factory/tier` and `factory/citations` required in branch
      protection, gated on `uncworks gate check` having run green on one real
      pull request
- [ ] 3.2 Confirm: the owner spot-checks that a deliberately non-conforming
      branch is blocked, and that the tier the gate assigned is the tier they
      would have assigned
