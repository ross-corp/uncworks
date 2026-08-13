# <Change title> tasks

## 1. <Phase name>

- **SHAPE** graph
- **MERGE** 1.3

- [ ] 1.1 <Task> `writes:` <paths|none>
- [ ] 1.2 <Task> `deps:` 1.1 `writes:` <paths|none>
- [ ] 1.3 Adversarial review of this phase against the proposal Behavior
      criteria. Run `uncworks spec check` and append a round block to review.md
      `deps:` 1.2

## 2. <Phase name>

- **SHAPE** loop
- **STOP** `task test:go` exits 0 and every new test fails on an injected defect
- **MAX-ITERS** 4
- **TERMINAL** CAPPED

- [ ] 2.1 <The task iteration 2 starts from>
- [ ] 2.2 Adversarial review of this phase. Run `uncworks spec check` and append
      a round block to review.md

## 3. Rollout

- [ ] 3.1 Apply: <impactful action>, gated on `<command>` exiting 0
- [ ] 3.2 Confirm: <the judgment only the owner can make>
