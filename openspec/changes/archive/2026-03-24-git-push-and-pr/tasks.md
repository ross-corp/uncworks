## 1. Enhanced PushChanges

- [x] 1.1 Add `DiffStat string` to `PushChangesOutput` — capture `git diff --stat HEAD~1` after commit
- [x] 1.2 Change `git push origin` to `git push --force origin` to handle existing branches
- [x] 1.3 Add test for DiffStat extraction

## 2. Enhanced PR Body

- [x] 2.1 In `postVerifyPushAndPR`: read `openspec/changes/{changeName}/proposal.md` via execInSidecar before CreatePR
- [x] 2.2 Include proposal content, diff stats, run metadata in the PR body markdown
- [x] 2.3 Add `DiffStat string` and `ProposalContent string` to `CreatePRInput`

## 3. Verification

- [ ] 3.1 Run a spec-driven run with autoPush=true and autoPR=true, verify branch and PR are created
- [ ] 3.2 Verify PR body contains proposal content and diff stats
