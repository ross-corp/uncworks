package gate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRepo answers from a fixed set, so a verdict test needs no git.
type fakeRepo struct {
	changed []string
	atBase  map[string]bool
}

func (f fakeRepo) ChangedPaths(context.Context, string, string) ([]string, error) {
	return f.changed, nil
}

func (f fakeRepo) Exists(_ context.Context, _, path string) (bool, error) {
	return f.atBase[path], nil
}

func TestClassifyTier(t *testing.T) {
	cases := []struct {
		name    string
		changed []string
		want    Tier
	}{
		{"documentation only", []string{"docs/guides/models.md", "README.md"}, TierT1},
		{"tests only", []string{"test/contract/boundary_test.go", "internal/gate/gate_test.go"}, TierT1},
		{"one capability", []string{"internal/gate/gate.go", "internal/gate/verdicts.go"}, TierT2},
		{"two capabilities", []string{"internal/gate/gate.go", "internal/cli/open.go"}, TierT3},
		{"the REST surface", []string{"internal/server/files.go"}, TierT3},
		{"a spec amendment", []string{"openspec/specs/run-pipeline/spec.md"}, TierAmendment},
		{"a change's own artifacts", []string{"openspec/changes/foo/proposal.md"}, TierT1},
		{"a proto change", []string{"proto/api.proto"}, TierT3},
		{"a deploy change", []string{"deploy/helm/aot/values.yaml"}, TierT3},
		{"an empty diff", nil, TierT1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := ClassifyTier(tc.changed)
			if got != tc.want {
				t.Fatalf("got %s (%s), want %s", got, reason, tc.want)
			}
			if reason == "" {
				t.Fatal("a tier with no stated reason is not arguable")
			}
		})
	}
}

func TestClassifyTier_IgnoresTheBranchName(t *testing.T) {
	// The classifier takes only paths. There is no branch input to pass, which
	// is the property under test: a change cannot lower its own tier by being
	// renamed.
	tier, _ := ClassifyTier([]string{"proto/api.proto"})
	if tier != TierT3 {
		t.Fatalf("a public-surface change is T3 whatever the branch is called, got %s", tier)
	}
}

func TestTierVerdict_T1NeedsNoSpec(t *testing.T) {
	repo := fakeRepo{changed: []string{"docs/README.md"}}
	result, err := Check(context.Background(), repo, Options{Base: "main", Head: "HEAD"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.Tier != TierT1 {
		t.Fatalf("tier: got %s", result.Tier)
	}
	if !verdict(result, "factory/tier").Pass {
		t.Fatal("a documentation change must pass with no spec")
	}
}

func TestTierVerdict_T2WithoutAMergedSpecFails(t *testing.T) {
	repo := fakeRepo{changed: []string{"internal/gate/gate.go"}, atBase: map[string]bool{}}
	result, err := Check(context.Background(), repo,
		Options{ChangeID: "factory-gate", Base: "main", Head: "HEAD", Root: t.TempDir()})
	if err == nil {
		t.Fatal("expected a required verdict to fail")
	}
	tier := verdict(result, "factory/tier")
	if tier.Pass {
		t.Fatal("expected the tier verdict to fail")
	}
	if !strings.Contains(strings.Join(tier.Detail, " "), "openspec/changes/factory-gate/proposal.md") {
		t.Fatalf("the verdict must name what it looked for, got %v", tier.Detail)
	}
}

func TestTierVerdict_NamesTheMissingChangeWhenNoneIsGiven(t *testing.T) {
	repo := fakeRepo{changed: []string{"internal/gate/gate.go"}}
	result, _ := Check(context.Background(), repo, Options{Base: "main", Head: "HEAD", Root: t.TempDir()})
	tier := verdict(result, "factory/tier")
	if tier.Pass {
		t.Fatal("a T2 change with no named change cannot pass")
	}
	if !strings.Contains(strings.Join(tier.Detail, " "), "--change") {
		t.Fatalf("the verdict must say how to fix it, got %v", tier.Detail)
	}
}

func TestTierVerdict_T3NeedsARecordedOwnerDecision(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "openspec", "changes", "wide")
	writeFile(t, filepath.Join(dir, "proposal.md"), "# P\n\n## Impact\n\n- proto/\n")
	writeFile(t, filepath.Join(dir, "review.md"), "# R\n\n## Owner decision\n\npending\n")
	writeFile(t, filepath.Join(dir, "citations.lock"), `{"records":[]}`)

	repo := fakeRepo{
		changed: []string{"proto/api.proto"},
		atBase:  map[string]bool{"openspec/changes/wide/proposal.md": true},
	}
	result, err := Check(context.Background(), repo,
		Options{ChangeID: "wide", Base: "main", Head: "HEAD", Root: root})
	if err == nil {
		t.Fatal("a T3 change with a pending decision must not pass")
	}
	if !strings.Contains(strings.Join(verdict(result, "factory/tier").Detail, " "), "pending") {
		t.Fatal("the verdict must say the decision is pending")
	}

	writeFile(t, filepath.Join(dir, "review.md"), "# R\n\n## Owner decision\n\napproved\n")
	result, err = Check(context.Background(), repo,
		Options{ChangeID: "wide", Base: "main", Head: "HEAD", Root: root})
	if err != nil {
		t.Fatalf("an approved T3 change must pass its tier verdict: %v", err)
	}
	if !verdict(result, "factory/tier").Pass {
		t.Fatal("expected the tier verdict to pass once the owner decided")
	}
}

func TestOrderVerdict_CodeBeforeItsSpecIsReported(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "openspec", "changes", "late", "proposal.md"),
		"# P\n\n## Impact\n\n- internal/gate/\n")

	repo := fakeRepo{changed: []string{"internal/gate/gate.go"}, atBase: map[string]bool{}}
	result, _ := Check(context.Background(), repo,
		Options{ChangeID: "late", Base: "main", Head: "HEAD", Root: root})

	order := verdict(result, "factory/order")
	if order.Pass {
		t.Fatal("code that arrives before its spec must be reported")
	}
	if order.Required {
		t.Fatal("order is advisory: a small change legitimately carries both at once")
	}
}

func TestOrderVerdict_ADocOnlyDiffPasses(t *testing.T) {
	repo := fakeRepo{changed: []string{"docs/x.md"}, atBase: map[string]bool{}}
	result, _ := Check(context.Background(), repo,
		Options{ChangeID: "late", Base: "main", Head: "HEAD", Root: t.TempDir()})
	if !verdict(result, "factory/order").Pass {
		t.Fatal("a diff with no code cannot precede its own spec")
	}
}

func TestConformance_NamesAnUndeclaredPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "openspec", "changes", "narrow", "proposal.md"),
		"# P\n\n## Impact\n\n- Code: `internal/gate/`\n")

	repo := fakeRepo{
		changed: []string{"internal/gate/gate.go", "internal/server/grpc.go"},
		atBase:  map[string]bool{"openspec/changes/narrow/proposal.md": true},
	}
	result, _ := Check(context.Background(), repo,
		Options{ChangeID: "narrow", Base: "main", Head: "HEAD", Root: root})

	conf := verdict(result, "factory/spec-conformance")
	if conf.Pass {
		t.Fatal("an undeclared path must fail the conformance verdict")
	}
	if conf.Required {
		t.Fatal("conformance is advisory")
	}
	if !strings.Contains(strings.Join(conf.Detail, " "), "internal/server/grpc.go") {
		t.Fatalf("the verdict must name the undeclared path, got %v", conf.Detail)
	}
}

func TestConformance_IgnoresGeneratedPaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "openspec", "changes", "narrow", "proposal.md"),
		"# P\n\n## Impact\n\n- Code: `internal/gate/`\n")

	repo := fakeRepo{
		changed: []string{"internal/gate/gate.go", "gen/go/api.pb.go", "web/package-lock.json"},
		atBase:  map[string]bool{"openspec/changes/narrow/proposal.md": true},
	}
	result, _ := Check(context.Background(), repo,
		Options{ChangeID: "narrow", Base: "main", Head: "HEAD", Root: root})

	if !verdict(result, "factory/spec-conformance").Pass {
		t.Fatalf("generated churn must not fail conformance: %v",
			verdict(result, "factory/spec-conformance").Detail)
	}
}

func TestConformance_ReportsADeclaredPathTheDiffNeverTouches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "openspec", "changes", "narrow", "proposal.md"),
		"# P\n\n## Impact\n\n- Code: `internal/gate/`, `internal/server/`\n")

	repo := fakeRepo{
		changed: []string{"internal/gate/gate.go"},
		atBase:  map[string]bool{"openspec/changes/narrow/proposal.md": true},
	}
	result, _ := Check(context.Background(), repo,
		Options{ChangeID: "narrow", Base: "main", Head: "HEAD", Root: root})

	detail := strings.Join(verdict(result, "factory/spec-conformance").Detail, " ")
	if !strings.Contains(detail, "declared but untouched: internal/server") {
		t.Fatalf("expected the untouched declaration to be reported, got %q", detail)
	}
}

func TestCitations_FailsOnATamperedSnapshot(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "openspec", "changes", "cited")
	writeFile(t, filepath.Join(dir, "proposal.md"), "# P\n\n## Impact\n\n- docs/\n")
	writeFile(t, filepath.Join(dir, "citations", "c.snapshot"), "the source text")
	writeFile(t, filepath.Join(dir, "citations", "c.snapshot.prov.json"),
		`{"url":"https://example.com","http_status":200,"sha256":"deadbeef"}`)
	writeFile(t, filepath.Join(dir, "citations.lock"),
		`{"records":[{"id":"c","source":"https://example.com","accessed":"2026-08-01",`+
			`"claim_class":"api","quote":"the source text","snapshot":"citations/c.snapshot",`+
			`"sha256":"deadbeef"}]}`)

	repo := fakeRepo{changed: []string{"docs/x.md"}}
	result, err := Check(context.Background(), repo,
		Options{ChangeID: "cited", Base: "main", Head: "HEAD", Root: root})
	if err == nil {
		t.Fatal("a mismatched sha256 must fail the required citations verdict")
	}
	cite := verdict(result, "factory/citations")
	if cite.Pass || !cite.Required {
		t.Fatalf("citations must be required and failing, got %+v", cite)
	}
}

func TestCitations_PassWhenNoChangeIsNamed(t *testing.T) {
	repo := fakeRepo{changed: []string{"docs/x.md"}}
	result, err := Check(context.Background(), repo, Options{Base: "main", Head: "HEAD", Root: t.TempDir()})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !verdict(result, "factory/citations").Pass {
		t.Fatal("with no change there is no lock to gate")
	}
}

func TestResult_CarriesEveryVerdict(t *testing.T) {
	repo := fakeRepo{changed: []string{"docs/x.md"}}
	result, _ := Check(context.Background(), repo, Options{Base: "main", Head: "HEAD", Root: t.TempDir()})
	for _, name := range []string{"factory/tier", "factory/citations", "factory/spec-conformance", "factory/order"} {
		if verdict(result, name).Name == "" {
			t.Fatalf("missing verdict %s", name)
		}
	}
}

func verdict(result Result, name string) Verdict {
	for _, v := range result.Verdicts {
		if v.Name == name {
			return v
		}
	}
	return Verdict{}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// The tests below each pin one objection from the round 1 adversarial review,
// so a regression reintroduces a named finding rather than a nameless bug.

func TestTierVerdict_ANamedChangeMustClaimTheDiff(t *testing.T) {
	// Round 1, objection 1, upheld by all three critics. The change id comes
	// from a branch name the author writes, so a diff could borrow any merged
	// change's spec and pass the required verdict.
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "openspec", "changes", "keybindings", "proposal.md"),
		"# P\n\n## Impact\n\n- Code: `web/src/hooks/`\n")

	repo := fakeRepo{
		changed: []string{"internal/cli/open.go"},
		atBase:  map[string]bool{"openspec/changes/keybindings/proposal.md": true},
	}
	result, err := Check(context.Background(), repo,
		Options{ChangeID: "keybindings", Base: "main", Head: "HEAD", Root: root})
	if err == nil {
		t.Fatal("a change whose Impact covers none of the diff must not pass the tier verdict")
	}
	detail := strings.Join(verdict(result, "factory/tier").Detail, " ")
	if !strings.Contains(detail, "does not claim this work") {
		t.Fatalf("the verdict must say the change does not claim the diff, got %q", detail)
	}
}

func TestTierVerdict_AClaimedDiffPasses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "openspec", "changes", "opener", "proposal.md"),
		"# P\n\n## Impact\n\n- Code: `internal/cli/`\n")

	repo := fakeRepo{
		changed: []string{"internal/cli/open.go"},
		atBase:  map[string]bool{"openspec/changes/opener/proposal.md": true},
	}
	writeFile(t, filepath.Join(root, "openspec", "changes", "opener", "citations.lock"), `{"records":[]}`)
	if _, err := Check(context.Background(), repo,
		Options{ChangeID: "opener", Base: "main", Head: "HEAD", Root: root}); err != nil {
		t.Fatalf("a change that claims its diff must pass: %v", err)
	}
}

func TestTierVerdict_ASpecAmendmentNeverPassesOnItsOwn(t *testing.T) {
	// Round 1, objection 4. openspec/ counted as documentation, so the one diff
	// shape the escalation protocol exists for was the one shape that could not
	// fail.
	repo := fakeRepo{changed: []string{"openspec/specs/run-pipeline/spec.md"}}
	result, err := Check(context.Background(), repo,
		Options{Base: "main", Head: "HEAD", Root: t.TempDir()})
	if err == nil {
		t.Fatal("an amendment to the spec corpus must not pass the tier verdict")
	}
	if result.Tier != TierAmendment {
		t.Fatalf("tier: got %s, want %s", result.Tier, TierAmendment)
	}
	if !strings.Contains(strings.Join(verdict(result, "factory/tier").Detail, " "), "escalation") {
		t.Fatal("the verdict must name the diff as an escalation")
	}
}

func TestCitations_AnAbsentLockFailsTheRequiredVerdict(t *testing.T) {
	// Round 1, objection from the bypass lens. citelock.Verify reads an absent
	// lock as nothing to do, which made the required verdict vacuous exactly
	// where the schema demands a lock.
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "openspec", "changes", "nolock", "proposal.md"),
		"# P\n\n## Impact\n\n- docs/\n")

	repo := fakeRepo{changed: []string{"docs/x.md"}}
	result, err := Check(context.Background(), repo,
		Options{ChangeID: "nolock", Base: "main", Head: "HEAD", Root: root})
	if err == nil {
		t.Fatal("a change with no citations.lock must fail the required citations verdict")
	}
	if !strings.Contains(verdict(result, "factory/citations").Summary, "no citations.lock") {
		t.Fatalf("the verdict must name the missing lock, got %q",
			verdict(result, "factory/citations").Summary)
	}
}

func TestResult_OnlyNarrowsToOneVerdict(t *testing.T) {
	// Round 1, objection 3. Branch protection needs one check name per verdict,
	// and a single job only ever gave it the job's own name.
	repo := fakeRepo{changed: []string{"docs/x.md"}}
	result, _ := Check(context.Background(), repo, Options{Base: "main", Head: "HEAD", Root: t.TempDir()})

	only := result.Only("factory/tier")
	if len(only.Verdicts) != 1 || only.Verdicts[0].Name != "factory/tier" {
		t.Fatalf("expected exactly the tier verdict, got %+v", only.Verdicts)
	}
	if len(result.Only("factory/nonexistent").Verdicts) != 0 {
		t.Fatal("an unknown name must narrow to nothing, so the caller can report it")
	}
}
