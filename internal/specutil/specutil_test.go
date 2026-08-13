package specutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goodChange is a change that satisfies every rule. Each test mutates one file
// so a failure names exactly one rule.
func goodChange(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "add-widget")
	write(t, dir, "proposal.md", `# Add widget

## Why

The widget is missing.

## What Changes

- Add a widget.

## Behavior

- B1: `+"`task test:go`"+` passes with the widget present.

## Impact

- internal/widget/
`)
	write(t, dir, "design.md", `# Design

## Context

Extends internal/thing.

## Decisions

- Decision: store the widget in PostgreSQL.
- Alternative rejected: an in-memory map, because it does not survive a restart.

## Rollout & Gating

Ship phase 1, then phase 2 once `+"`task test:go`"+` exits 0.

## Risks / Trade-offs

- A slow query, mitigated by an index.
`)
	write(t, dir, "tasks.md", `# Tasks

## 1. Build the widget

- **SHAPE** graph
- **MERGE** 1.3

- [ ] 1.1 Add the widget type `+"`writes:` internal/widget/type.go"+`
- [ ] 1.2 Add the widget store `+"`deps:` 1.1 `writes:` internal/widget/store.go"+`
- [ ] 1.3 Adversarial review of the widget phase `+"`deps:` 1.2"+`

## 2. Rollout

- [ ] 2.1 Apply: deploy the widget, gated on `+"`task test:go`"+` exiting 0
- [ ] 2.2 Confirm: the owner spot-checks that the stored widget is the one they meant
`)
	write(t, dir, "review.md", `# Review

## Rubric

The proposal Behavior criteria and the design Decisions.

## Deterministic lint

`+"`uncworks spec check`"+` exits 0.

## Recommendation

Run the critic loop, because the store is new.

## Owner decision

approved
`)
	write(t, dir, "citations.lock", `{"records":[]}`)
	return dir
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func checkDir(t *testing.T, dir string) Report {
	t.Helper()
	change, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return Check(change)
}

// rules returns the set of rule ids that fired at error severity.
func rules(report Report) map[string]string {
	out := map[string]string{}
	for _, f := range report.Findings {
		if f.Severity == SeverityError {
			out[f.Rule] = f.Message
		}
	}
	return out
}

func TestCheck_AConformingChangePasses(t *testing.T) {
	report := checkDir(t, goodChange(t))
	if report.Failed() {
		t.Fatalf("expected a clean report, got %v", rules(report))
	}
}

func TestCheck_MissingBehaviorSectionFails(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "proposal.md"))
	write(t, dir, "proposal.md", strings.Replace(string(body), "## Behavior", "## Criteria", 1))

	if _, ok := rules(checkDir(t, dir))["proposal-sections"]; !ok {
		t.Fatal("expected proposal-sections to fire")
	}
}

func TestCheck_EmptyBehaviorSectionFails(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "proposal.md", "# P\n\n## Why\n\nx\n\n## What Changes\n\n- y\n\n## Behavior\n\n## Impact\n\n- z\n")

	if _, ok := rules(checkDir(t, dir))["behavior-has-criteria"]; !ok {
		t.Fatal("expected behavior-has-criteria to fire on an empty Behavior section")
	}
}

func TestCheck_DecisionWithoutRejectedAlternativeFails(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "design.md"))
	stripped := strings.Replace(string(body),
		"- Alternative rejected: an in-memory map, because it does not survive a restart.\n", "", 1)
	write(t, dir, "design.md", stripped)

	if _, ok := rules(checkDir(t, dir))["decision-has-alternative"]; !ok {
		t.Fatal("expected decision-has-alternative to fire")
	}
}

func TestCheck_PendingOwnerDecisionFails(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "review.md"))
	write(t, dir, "review.md", strings.Replace(string(body), "approved", "pending", 1))

	if _, ok := rules(checkDir(t, dir))["review-decision-current"]; !ok {
		t.Fatal("expected review-decision-current to fire on a pending decision")
	}
}

func TestCheck_PhaseWithoutShapeFails(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "tasks.md"))
	write(t, dir, "tasks.md", strings.Replace(string(body), "- **SHAPE** graph\n", "", 1))

	if _, ok := rules(checkDir(t, dir))["phase-shape-declared"]; !ok {
		t.Fatal("expected phase-shape-declared to fire")
	}
}

func TestCheck_LoopWithoutBothBoundsFails(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "tasks.md", "# Tasks\n\n## 1. Converge\n\n- **SHAPE** loop\n- **STOP** `task test:go` exits 0\n\n"+
		"- [ ] 1.1 Fix a failure\n- [ ] 1.2 Adversarial review\n")

	got := rules(checkDir(t, dir))
	if _, ok := got["loop-markers"]; !ok {
		t.Fatalf("expected loop-markers to fire for the missing MAX-ITERS and TERMINAL, got %v", got)
	}
}

func TestCheck_ProseStopIsRejected(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "tasks.md", "# Tasks\n\n## 1. Converge\n\n- **SHAPE** loop\n"+
		"- **STOP** the settings are coherent\n- **MAX-ITERS** 4\n- **TERMINAL** CAPPED\n\n"+
		"- [ ] 1.1 Fix a failure\n- [ ] 1.2 Adversarial review\n")

	if _, ok := rules(checkDir(t, dir))["stop-is-a-command"]; !ok {
		t.Fatal("expected stop-is-a-command to fire on a prose STOP")
	}
}

func TestCheck_BacktickedFileIsNotACommand(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "tasks.md", "# Tasks\n\n## 1. Converge\n\n- **SHAPE** loop\n"+
		"- **STOP** `internal/widget/store.go` passes its fixtures\n- **MAX-ITERS** 4\n"+
		"- **TERMINAL** CAPPED\n\n- [ ] 1.1 Fix a failure\n- [ ] 1.2 Adversarial review\n")

	if _, ok := rules(checkDir(t, dir))["stop-is-a-command"]; !ok {
		t.Fatal("a backtick span naming a file is not a command, so the rule must fire")
	}
}

func TestCheck_GraphWithoutMergeFails(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "tasks.md"))
	write(t, dir, "tasks.md", strings.Replace(string(body), "- **MERGE** 1.3\n", "", 1))

	if _, ok := rules(checkDir(t, dir))["graph-merge-declared"]; !ok {
		t.Fatal("expected graph-merge-declared to fire")
	}
}

func TestCheck_PhaseWithoutAdversarialReviewFails(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "tasks.md"))
	write(t, dir, "tasks.md", strings.Replace(string(body),
		"- [ ] 1.3 Adversarial review of the widget phase `deps:` 1.2", "- [ ] 1.3 Tidy up `deps:` 1.2", 1))

	if _, ok := rules(checkDir(t, dir))["phase-has-review"]; !ok {
		t.Fatal("expected phase-has-review to fire")
	}
}

func TestCheck_UnresolvableDependencyFails(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "tasks.md"))
	write(t, dir, "tasks.md", strings.Replace(string(body), "`deps:` 1.1", "`deps:` 2.9", 1))

	if _, ok := rules(checkDir(t, dir))["task-deps-resolve"]; !ok {
		t.Fatal("expected task-deps-resolve to fire on a cross-phase edge")
	}
}

func TestCheck_DependencyCycleFails(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "tasks.md", "# Tasks\n\n## 1. Build\n\n- **SHAPE** graph\n- **MERGE** 1.3\n\n"+
		"- [ ] 1.1 A `deps:` 1.2\n- [ ] 1.2 B `deps:` 1.1\n- [ ] 1.3 Adversarial review `deps:` 1.2\n")

	got := rules(checkDir(t, dir))
	if _, ok := got["task-deps-acyclic"]; !ok {
		t.Fatalf("expected task-deps-acyclic to fire, got %v", got)
	}
}

func TestCheck_EmDashIsRejected(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "proposal.md"))
	write(t, dir, "proposal.md", strings.Replace(string(body),
		"The widget is missing.", "The widget is missing — nobody added it.", 1))

	if _, ok := rules(checkDir(t, dir))["no-em-dash"]; !ok {
		t.Fatal("expected no-em-dash to fire")
	}
}

func TestCheck_BoldedBulletLeadIsRejected(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "proposal.md"))
	write(t, dir, "proposal.md", strings.Replace(string(body),
		"- Add a widget.", "- **Schedule.** Add a widget.", 1))

	if _, ok := rules(checkDir(t, dir))["bolded-bullet-lead"]; !ok {
		t.Fatal("expected bolded-bullet-lead to fire")
	}
}

func TestCheck_MarkerLabelsMayLeadABullet(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "proposal.md"))
	write(t, dir, "proposal.md", strings.Replace(string(body),
		"- Add a widget.", "- **BREAKING** the old widget goes away.", 1))

	if _, ok := rules(checkDir(t, dir))["bolded-bullet-lead"]; ok {
		t.Fatal("a declared marker label is allowed to lead a bullet")
	}
}

func TestCheck_MissingCitationsLockFails(t *testing.T) {
	dir := goodChange(t)
	if err := os.Remove(filepath.Join(dir, "citations.lock")); err != nil {
		t.Fatalf("remove: %v", err)
	}

	if _, ok := rules(checkDir(t, dir))["citations-lock-present"]; !ok {
		t.Fatal("expected citations-lock-present to fire")
	}
}

func TestCheck_DanglingCitationFails(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "proposal.md"))
	write(t, dir, "proposal.md", strings.Replace(string(body),
		"The widget is missing.", "The widget is missing [cite: nowhere].", 1))

	if _, ok := rules(checkDir(t, dir))["citation-dangling"]; !ok {
		t.Fatal("expected citation-dangling to fire")
	}
}

func TestCheck_UncitedRecordFails(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "citations.lock",
		`{"records":[{"id":"orphan","source":"https://example.com","quote":"q","snapshot":"s","sha256":"h","accessed":"2026-01-01","claim_class":"api"}]}`)

	if _, ok := rules(checkDir(t, dir))["citation-uncited"]; !ok {
		t.Fatal("expected citation-uncited to fire on a record nothing references")
	}
}

func TestCheck_RequirementWithoutNegativeScenarioFails(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "specs/widget/spec.md", `## ADDED Requirements

### Requirement: The widget MUST persist

#### Scenario: Widget survives a restart
- **POLARITY** positive
- **WHEN** the process restarts
- **THEN** the widget is still stored
`)

	if _, ok := rules(checkDir(t, dir))["spec-negative-scenario"]; !ok {
		t.Fatal("expected spec-negative-scenario to fire")
	}
}

func TestCheck_PolarityIsReadFromTheMarkerNotTheProse(t *testing.T) {
	dir := goodChange(t)
	// The THEN prose carries a failure token, and the declared polarity says
	// positive. The lint must follow the marker.
	write(t, dir, "specs/widget/spec.md", `## ADDED Requirements

### Requirement: The widget MUST persist

#### Scenario: Widget survives a restart
- **POLARITY** positive
- **THEN** the command exits non-zero on a missing widget

#### Scenario: Widget is absent
- **POLARITY** negative
- **THEN** the read succeeds and returns nothing
`)

	got := rules(checkDir(t, dir))
	if _, ok := got["spec-negative-scenario"]; ok {
		t.Fatalf("the declared negative scenario satisfies the rule, got %v", got)
	}
	if _, ok := got["scenario-polarity"]; ok {
		t.Fatalf("both scenarios declare a polarity, got %v", got)
	}
}

func TestCheck_UndeclaredPolarityFails(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "specs/widget/spec.md", `## ADDED Requirements

### Requirement: The widget MUST persist

#### Scenario: Widget survives
- **WHEN** the process restarts
- **THEN** the widget is still stored
`)

	if _, ok := rules(checkDir(t, dir))["scenario-polarity"]; !ok {
		t.Fatal("expected scenario-polarity to fire")
	}
}

func TestCheck_MultiCapabilityChangeNeedsNonGoals(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "specs/widget/spec.md", "### Requirement: A\n\n#### Scenario: S\n- **POLARITY** negative\n")
	write(t, dir, "specs/gadget/spec.md", "### Requirement: B\n\n#### Scenario: S\n- **POLARITY** negative\n")

	if _, ok := rules(checkDir(t, dir))["proposal-non-goals"]; !ok {
		t.Fatal("expected proposal-non-goals to fire for a two-capability change")
	}
}

func TestNext_ReportsTheReadySetInDependencyOrder(t *testing.T) {
	dir := goodChange(t)
	change, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	ready, err := Next(change)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if ready.Phase != 1 || ready.Shape != "graph" {
		t.Fatalf("expected phase 1 graph, got phase %d shape %q", ready.Phase, ready.Shape)
	}
	if len(ready.Tasks) != 1 || ready.Tasks[0].ID != "1.1" {
		t.Fatalf("expected only 1.1 to be ready, got %+v", ready.Tasks)
	}
	if len(ready.Blocked) != 2 {
		t.Fatalf("expected 1.2 and 1.3 to be blocked, got %+v", ready.Blocked)
	}
	if ready.Concurrent {
		t.Fatal("one ready task is not concurrent work")
	}
}

func TestNext_SkipsAFinishedPhase(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "tasks.md"))
	write(t, dir, "tasks.md", strings.ReplaceAll(string(body), "- [ ] 1.", "- [x] 1."))

	change, _ := Load(dir)
	ready, err := Next(change)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if ready.Phase != 2 {
		t.Fatalf("expected phase 2, got %d", ready.Phase)
	}
}

func TestNext_ALoopPhaseIsNeverConcurrent(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "tasks.md", "# Tasks\n\n## 1. Converge\n\n- **SHAPE** loop\n"+
		"- **STOP** `task test:go` exits 0\n- **MAX-ITERS** 4\n- **TERMINAL** CAPPED\n\n"+
		"- [ ] 1.1 Fix a failure\n- [ ] 1.2 Fix another failure\n- [ ] 1.3 Adversarial review\n")

	change, _ := Load(dir)
	ready, _ := Next(change)
	if ready.Concurrent {
		t.Fatal("a loop's next iteration reads what the current one wrote, so it is never concurrent")
	}
}

func TestNext_AGateIsNeverConcurrent(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "tasks.md", "# Tasks\n\n## 1. Ship\n\n- **SHAPE** graph\n- **MERGE** 1.3\n\n"+
		"- [ ] 1.1 Write the code\n- [ ] 1.2 Apply: `git push`, gated on `task test:go` exiting 0\n"+
		"- [ ] 1.3 Adversarial review\n")

	change, _ := Load(dir)
	ready, _ := Next(change)
	if ready.Concurrent {
		t.Fatal("an owner gate must never be fanned out")
	}
}

func TestNext_ReportsACycleRatherThanStalling(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "tasks.md", "# Tasks\n\n## 1. Build\n\n- **SHAPE** graph\n- **MERGE** 1.2\n\n"+
		"- [ ] 1.1 A `deps:` 1.2\n- [ ] 1.2 Adversarial review `deps:` 1.1\n")

	change, _ := Load(dir)
	if _, err := Next(change); err == nil {
		t.Fatal("expected a cycle error rather than an empty ready set")
	}
}

func TestWriteConflicts_FindsAnIntersectingWriteSet(t *testing.T) {
	dir := goodChange(t)
	write(t, dir, "tasks.md", "# Tasks\n\n## 1. Build\n\n- **SHAPE** graph\n- **MERGE** 1.3\n\n"+
		"- [ ] 1.1 A `writes:` internal/widget/store.go\n"+
		"- [ ] 1.2 B `writes:` internal/widget/store.go\n"+
		"- [ ] 1.3 Adversarial review `deps:` 1.1, 1.2\n")

	change, _ := Load(dir)
	ready, _ := Next(change)
	conflicts := WriteConflicts(ready)
	if len(conflicts["internal/widget/store.go"]) != 2 {
		t.Fatalf("expected 1.1 and 1.2 to collide, got %v", conflicts)
	}
}

func TestParseTask_ReadsIDDepsAndWrites(t *testing.T) {
	task := parseTask("1.4 Add the store `deps:` 1.1, 1.2 `writes:` a.go, b.go", false, 7)
	if task.ID != "1.4" {
		t.Fatalf("id: got %q", task.ID)
	}
	if task.Text != "Add the store" {
		t.Fatalf("text: got %q", task.Text)
	}
	if len(task.Deps) != 2 || task.Deps[0] != "1.1" {
		t.Fatalf("deps: got %v", task.Deps)
	}
	if len(task.Writes) != 2 || task.Writes[1] != "b.go" {
		t.Fatalf("writes: got %v", task.Writes)
	}
}

func TestParseTask_NoneMeansAnEmptyList(t *testing.T) {
	task := parseTask("1.1 Start `deps:` none", false, 1)
	if task.Deps == nil || len(task.Deps) != 0 {
		t.Fatalf("expected an empty declared list, got %v", task.Deps)
	}
}

func TestStatus_CountsCheckedTasks(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "tasks.md"))
	write(t, dir, "tasks.md", strings.Replace(string(body), "- [ ] 1.1", "- [x] 1.1", 1))

	change, _ := Load(dir)
	progress := Status(change)
	if progress.Done != 1 || progress.Total != 5 {
		t.Fatalf("got %d/%d", progress.Done, progress.Total)
	}
}

func TestMermaid_RendersEveryPhaseAndEdge(t *testing.T) {
	change, _ := Load(goodChange(t))
	out := Mermaid(change)
	for _, want := range []string{"flowchart TD", "subgraph p1", "subgraph p2", "t1_1 --> t1_2"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in:\n%s", want, out)
		}
	}
}

func TestCheck_IgnoresAFencedCodeBlock(t *testing.T) {
	dir := goodChange(t)
	body, _ := os.ReadFile(filepath.Join(dir, "design.md"))
	write(t, dir, "design.md", string(body)+"\n```\n- **Label.** an em-dash — inside a fence\n```\n")

	got := rules(checkDir(t, dir))
	if _, ok := got["no-em-dash"]; ok {
		t.Fatalf("a fenced block is a sample, not prose, got %v", got)
	}
	if _, ok := got["bolded-bullet-lead"]; ok {
		t.Fatalf("a fenced block is a sample, not prose, got %v", got)
	}
}
