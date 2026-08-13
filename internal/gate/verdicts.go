package gate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/uncworks/aot/internal/citelock"
	"github.com/uncworks/aot/internal/specutil"
)

// changeDir is where a change's artifacts live, relative to the repo root.
func changeDir(root, id string) string {
	return filepath.Join(root, "openspec", "changes", id)
}

// changePath is the same location as a repo-relative git path.
func changePath(id string) string {
	return "openspec/changes/" + id
}

// tierVerdict enforces what the computed tier demands.
//
// It requires the spec to be present, not to be perfect. The changes that were
// in flight when this landed predate the schema and do not pass the rubric
// lint, and blocking them here would gate old work on a rule written after it.
func tierVerdict(ctx context.Context, repo Repo, opts Options, tier Tier, reason string, changed []string) Verdict {
	verdict := Verdict{Name: "factory/tier", Required: true, Summary: string(tier) + ": " + reason}

	switch tier {
	case TierExperiment, TierT1:
		verdict.Pass = true
		return verdict
	case TierAmendment:
		// An amendment rewrites the contract other work is measured against, so
		// it never passes on its own. The owner merges it as its own change,
		// and the code that depends on it comes afterwards.
		verdict.Detail = append(verdict.Detail,
			"this diff amends openspec/specs/, so it is an escalation rather than an "+
				"implementation. Merge the amendment on its own, then reopen the code "+
				"against the amended spec")
		return verdict
	}

	if opts.ChangeID == "" {
		verdict.Detail = append(verdict.Detail,
			"tier "+string(tier)+" needs a merged spec, and the pull request names no change. "+
				"Name it with --change, or in a branch of the form change/<id>")
		return verdict
	}

	proposal := changePath(opts.ChangeID) + "/proposal.md"
	present, err := repo.Exists(ctx, opts.Base, proposal)
	if err != nil {
		verdict.Detail = append(verdict.Detail, "could not read the base ref: "+err.Error())
		return verdict
	}
	if !present {
		verdict.Detail = append(verdict.Detail,
			fmt.Sprintf("tier %s needs a merged spec, and %s is absent from %s",
				tier, proposal, opts.Base))
		return verdict
	}

	// The change id comes from a flag or a branch name, both of which the
	// author writes. Checking only that the named change exists lets any diff
	// borrow any merged change's spec, which is the same bar-lowering the tier
	// classifier refuses to allow through a rename. Require the named change to
	// have claimed this work.
	change, err := specutil.Load(changeDir(opts.Root, opts.ChangeID))
	if err != nil {
		verdict.Detail = append(verdict.Detail, "could not read the change: "+err.Error())
		return verdict
	}
	if unclaimed := unclaimedPaths(change, changed); len(unclaimed) > 0 {
		verdict.Detail = append(verdict.Detail,
			fmt.Sprintf("the change %q does not claim this work: its Impact covers none of %s",
				opts.ChangeID, strings.Join(truncateList(unclaimed, 5), ", ")))
		return verdict
	}

	if tier == TierT3 {
		change, err := specutil.Load(changeDir(opts.Root, opts.ChangeID))
		if err != nil {
			verdict.Detail = append(verdict.Detail, "could not read the change: "+err.Error())
			return verdict
		}
		review := change.Artifacts["review"]
		if review == nil {
			verdict.Detail = append(verdict.Detail, "tier T3 needs a review.md, and the change has none")
			return verdict
		}
		decision := review.Section("## Owner decision")
		if decision == nil {
			verdict.Detail = append(verdict.Detail, "review.md records no owner decision")
			return verdict
		}
		body := strings.ToLower(strings.Join(decision.Body, " "))
		if strings.Contains(body, "pending") || strings.TrimSpace(body) == "" {
			verdict.Detail = append(verdict.Detail,
				"the owner decision is still pending, so a tier T3 change cannot merge")
			return verdict
		}
	}

	verdict.Pass = true
	return verdict
}

// unclaimedPaths returns the changed code paths the change's Impact does not
// cover. It returns nothing when at least one path is claimed, because a
// partial declaration is the conformance check's business, not the tier's. The
// tier only asks whether this change is about this diff at all.
func unclaimedPaths(change *specutil.Change, changed []string) []string {
	declared := declaredPaths(change)
	var code, unclaimed []string
	for _, p := range changed {
		if isGenerated(p) || isChangeArtifact(p) || isDoc(p) || isTest(p) {
			continue
		}
		code = append(code, p)
		if !coveredBy(declared, p) {
			unclaimed = append(unclaimed, p)
		}
	}
	if len(code) == 0 || len(unclaimed) < len(code) {
		return nil
	}
	return unclaimed
}

func truncateList(items []string, n int) []string {
	if len(items) <= n {
		return items
	}
	return append(items[:n:n], fmt.Sprintf("and %d more", len(items)-n))
}

// citationsVerdict runs the offline gate over the change's pinned claims.
//
// It performs no live fetch. A live fetch would make the verdict a function of
// network state, so an untouched pull request would start failing on its own.
func citationsVerdict(opts Options) Verdict {
	verdict := Verdict{Name: "factory/citations", Required: true}
	if opts.ChangeID == "" {
		verdict.Pass = true
		verdict.Summary = "no change named, so there is no lock to gate"
		return verdict
	}

	dir := changeDir(opts.Root, opts.ChangeID)
	if _, err := os.Stat(dir); err != nil {
		verdict.Pass = true
		verdict.Summary = "the change directory is not in this tree"
		return verdict
	}
	// citelock.Verify treats an absent lock as nothing to do, which is right
	// for the library and wrong here: the schema requires every change to carry
	// one, so an absent lock cannot be read as "there was nothing to cite".
	if _, err := os.Stat(filepath.Join(dir, "citations.lock")); err != nil {
		verdict.Summary = "the change carries no citations.lock. Write {\"records\":[]} " +
			"to state that it cites nothing"
		return verdict
	}

	var findings []string
	err := citelock.Verify(dir, func(format string, args ...any) {
		findings = append(findings, fmt.Sprintf(format, args...))
	})
	if err != nil {
		verdict.Summary = err.Error()
		verdict.Detail = findings
		return verdict
	}
	verdict.Pass = true
	verdict.Summary = "the offline citation gate passed"
	return verdict
}

// conformanceVerdict compares the changed paths against the paths the change
// declares in its proposal's Impact section.
func conformanceVerdict(opts Options, changed []string) Verdict {
	verdict := Verdict{Name: "factory/spec-conformance", Required: false}
	if opts.ChangeID == "" {
		verdict.Pass = true
		verdict.Summary = "no change named, so there is nothing to compare against"
		return verdict
	}

	change, err := specutil.Load(changeDir(opts.Root, opts.ChangeID))
	if err != nil {
		verdict.Summary = "could not read the change: " + err.Error()
		return verdict
	}
	declared := declaredPaths(change)
	if len(declared) == 0 {
		verdict.Summary = "the proposal's Impact section declares no path"
		return verdict
	}

	var undeclared, untouched []string
	for _, p := range changed {
		if isGenerated(p) || isChangeArtifact(p) {
			continue
		}
		if !coveredBy(declared, p) {
			undeclared = append(undeclared, p)
		}
	}
	for _, d := range declared {
		if !touches(changed, d) {
			untouched = append(untouched, d)
		}
	}

	for _, p := range undeclared {
		verdict.Detail = append(verdict.Detail, "undeclared: "+p)
	}
	for _, d := range untouched {
		verdict.Detail = append(verdict.Detail, "declared but untouched: "+d)
	}
	verdict.Pass = len(undeclared) == 0
	verdict.Summary = fmt.Sprintf("%d undeclared paths, %d declared paths untouched",
		len(undeclared), len(untouched))
	return verdict
}

// orderVerdict confirms the spec merged before the code that implements it.
func orderVerdict(ctx context.Context, repo Repo, opts Options, changed []string) Verdict {
	verdict := Verdict{Name: "factory/order", Required: false}
	if opts.ChangeID == "" {
		verdict.Pass = true
		verdict.Summary = "no change named"
		return verdict
	}

	code := false
	for _, p := range changed {
		if !isDoc(p) && !isChangeArtifact(p) {
			code = true
			break
		}
	}
	if !code {
		verdict.Pass = true
		verdict.Summary = "the diff changes no code, so nothing can precede its spec"
		return verdict
	}

	proposal := changePath(opts.ChangeID) + "/proposal.md"
	present, err := repo.Exists(ctx, opts.Base, proposal)
	if err != nil {
		verdict.Summary = "could not read the base ref: " + err.Error()
		return verdict
	}
	if !present {
		verdict.Summary = fmt.Sprintf("the code changes but %s is absent from %s, "+
			"so the spec did not merge first", proposal, opts.Base)
		return verdict
	}
	verdict.Pass = true
	verdict.Summary = "the spec is present in " + opts.Base
	return verdict
}

// declaredPaths reads the path-like entries out of the proposal's Impact
// section. A bullet may name several paths, and prose around them is ignored.
func declaredPaths(change *specutil.Change) []string {
	proposal := change.Artifacts["proposal"]
	if proposal == nil {
		return nil
	}
	impact := proposal.Section("## Impact")
	if impact == nil {
		return nil
	}
	var paths []string
	for _, bullet := range impact.Bullets() {
		for _, field := range strings.FieldsFunc(bullet, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '`'
		}) {
			field = strings.Trim(field, ".:;()")
			if looksLikePath(field) {
				paths = append(paths, field)
			}
		}
	}
	sort.Strings(paths)
	return dedupe(paths)
}

// looksLikePath keeps the tokens that name a file or a directory in this
// repository, and drops the prose around them.
func looksLikePath(token string) bool {
	if token == "" || strings.ContainsAny(token, " \t") {
		return false
	}
	if !strings.Contains(token, "/") && !strings.Contains(token, ".") {
		return false
	}
	// A sentence ending in a filename-looking word, such as "revertible.", is
	// not a path. Require a separator or a known source extension.
	if strings.Contains(token, "/") {
		return true
	}
	for _, ext := range []string{".go", ".ts", ".tsx", ".yml", ".yaml", ".json", ".proto", ".md"} {
		if strings.HasSuffix(token, ext) {
			return true
		}
	}
	return false
}

// coveredBy reports whether a declared entry covers a changed path. A declared
// directory covers everything under it.
func coveredBy(declared []string, p string) bool {
	for _, d := range declared {
		d = strings.TrimSuffix(d, "/")
		if p == d || strings.HasPrefix(p, d+"/") {
			return true
		}
	}
	return false
}

func touches(changed []string, declared string) bool {
	declared = strings.TrimSuffix(declared, "/")
	for _, p := range changed {
		if p == declared || strings.HasPrefix(p, declared+"/") {
			return true
		}
	}
	return false
}

// isGenerated marks a path nobody declares and whose churn would drown the
// conformance signal.
func isGenerated(p string) bool {
	for _, prefix := range []string{"gen/", "cmd/uncworks-app/frontend/wailsjs/"} {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return strings.HasSuffix(p, "zz_generated.deepcopy.go") ||
		strings.HasSuffix(p, "package-lock.json")
}

// isChangeArtifact marks the change's own files, which are never part of the
// diff a conformance check is about.
func isChangeArtifact(p string) bool {
	return strings.HasPrefix(p, "openspec/")
}
