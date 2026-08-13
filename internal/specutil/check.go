package specutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Sentinel errors for the package.
var (
	// ErrNotAChange reports a path that is not a change directory.
	ErrNotAChange = errors.New("not a change directory")
	// ErrRubric reports that at least one rule failed at error severity.
	ErrRubric = errors.New("rubric check failed")
	// ErrCycle reports that the declared dependencies form a cycle.
	ErrCycle = errors.New("task dependencies form a cycle")
)

// Severity is how loudly a finding speaks.
type Severity string

const (
	// SeverityError fails the check.
	SeverityError Severity = "error"
	// SeverityWarn is reported and does not fail the check.
	SeverityWarn Severity = "warn"
)

// Finding is one rule violation.
type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Message  string   `json:"message"`
}

// Report is the result of checking one change.
type Report struct {
	Change   string    `json:"change"`
	Findings []Finding `json:"findings"`
}

// Failed reports whether any finding is an error.
func (r Report) Failed() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Rules is the rubric, in the order it is reported. Each entry states only
// facts the author wrote down. No rule infers intent, because a rule that
// guesses produces a verdict nobody can act on.
var Rules = []struct {
	ID   string
	What string
}{
	{"proposal-sections", "proposal.md has Why, What Changes, Behavior, and Impact"},
	{"proposal-non-goals", "a change spanning capabilities declares Non-goals"},
	{"behavior-has-criteria", "the Behavior section states at least one criterion"},
	{"design-sections", "design.md has Context, Decisions, Rollout & Gating, and Risks"},
	{"decision-has-alternative", "every Decision names a rejected alternative"},
	{"review-exists", "review.md exists and records a rubric and an owner decision"},
	{"review-decision-current", "the owner decision is not pending"},
	{"phase-shape-declared", "every non-rollout phase declares SHAPE loop or graph"},
	{"loop-markers", "every loop phase declares STOP, MAX-ITERS, and TERMINAL"},
	{"stop-is-a-command", "a loop's STOP opens a backtick span with a command"},
	{"graph-merge-declared", "a graph phase with more than one task declares MERGE"},
	{"phase-has-review", "every non-rollout phase ends with an adversarial-review task"},
	{"task-id-required", "every task carries an N.M identifier"},
	{"task-id-matches-phase", "a task's identifier belongs to its own phase"},
	{"task-deps-resolve", "every deps reference names a task in the same phase"},
	{"task-deps-acyclic", "the declared dependencies form no cycle"},
	{"spec-negative-scenario", "every requirement has a declared-negative scenario"},
	{"scenario-polarity", "every scenario declares POLARITY positive or negative"},
	{"citations-lock-present", "the change carries a citations.lock, even an empty one"},
	{"citation-uncited", "every citations.lock record is cited in prose"},
	{"citation-dangling", "every [cite: id] names a record that exists"},
	{"no-em-dash", "prose uses no em-dash"},
	{"bolded-bullet-lead", "no list item opens with a bolded label"},
}

// stopCommandRE requires a STOP to open a backtick span with something
// runnable. Any backtick span is too weak: "`internal/x.go` compiles" names a
// file, not a command, and nothing can evaluate it.
var stopCommandRE = regexp.MustCompile(
	"`(task|go|golangci-lint|gofmt|buf|npm|npx|kubectl|helm|docker|openspec|uncworks|" +
		"git|jq|bash|sh|make|python3?|devbox)\\b[^`]*`")

// allowedBoldLeads are the marker labels a list item may open with. Everything
// else reads as a bolded prefix doing work a plain sentence should do.
var allowedBoldLeads = map[string]bool{
	"WHEN": true, "THEN": true, "AND": true, "POLARITY": true,
	"SHAPE": true, "STOP": true, "MAX-ITERS": true, "TERMINAL": true,
	"MERGE": true, "BREAKING": true,
}

var (
	rolloutRE  = regexp.MustCompile(`(?i)rollout`)
	reviewRE   = regexp.MustCompile(`(?i)adversarial\s+review`)
	boldLeadRE = regexp.MustCompile(`^\s*[-*]\s+\*\*([^*]+)\*\*`)
	citeRE     = regexp.MustCompile(`\[cite:\s*([^\]]+)\]`)
)

// Check runs the rubric over a change. It is a pure function of the files on
// disk, so the same tree always yields the same findings.
func Check(change *Change) Report {
	report := Report{Change: change.Name}
	add := func(rule string, sev Severity, file string, line int, format string, args ...any) {
		report.Findings = append(report.Findings, Finding{
			Rule: rule, Severity: sev, File: file, Line: line,
			Message: fmt.Sprintf(format, args...),
		})
	}

	checkProposal(change, add)
	checkDesign(change, add)
	checkReview(change, add)
	checkPhases(change, add)
	checkTaskGraph(change, add)
	checkSpecs(change, add)
	checkCitations(change, add)
	checkProse(change, add)
	return report
}

type adder func(rule string, sev Severity, file string, line int, format string, args ...any)

func checkProposal(change *Change, add adder) {
	proposal := change.Artifacts["proposal"]
	if proposal == nil {
		add("proposal-sections", SeverityError, "proposal.md", 0,
			"the change has no proposal.md")
		return
	}
	for _, heading := range []string{"## Why", "## What Changes", "## Behavior", "## Impact"} {
		if proposal.Section(heading) == nil {
			add("proposal-sections", SeverityError, proposal.Path, 0,
				"proposal.md has no %q section", heading)
		}
	}

	behavior := proposal.Section("## Behavior")
	if behavior != nil && len(behavior.Bullets()) == 0 {
		add("behavior-has-criteria", SeverityError, proposal.Path, behavior.Line,
			"the Behavior section states no criterion. It is the rubric every later "+
				"artifact is reviewed against, so an empty one leaves the review nothing to bind to")
	}

	// Non-goals is required once a change touches more than one capability,
	// because that is when the boundary stops being obvious.
	if len(specDirs(change)) > 1 && proposal.Section("### Non-goals") == nil {
		add("proposal-non-goals", SeverityError, proposal.Path, 0,
			"the change touches %d capabilities and declares no Non-goals", len(specDirs(change)))
	}
}

func checkDesign(change *Change, add adder) {
	design := change.Artifacts["design"]
	if design == nil {
		add("design-sections", SeverityError, "design.md", 0, "the change has no design.md")
		return
	}
	for _, heading := range []string{"## Context", "## Decisions", "## Rollout & Gating", "## Risks / Trade-offs"} {
		if design.Section(heading) == nil {
			add("design-sections", SeverityError, design.Path, 0,
				"design.md has no %q section", heading)
		}
	}

	// Every `- Decision:` must be followed by an `- Alternative rejected:`. A
	// decision with no recorded rejected alternative is an assumption.
	decisions := design.Section("## Decisions")
	if decisions == nil {
		return
	}
	var pending string
	for offset, line := range decisions.Body {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "- Decision:"):
			if pending != "" {
				add("decision-has-alternative", SeverityError, design.Path, decisions.Line+offset,
					"the decision %q records no rejected alternative", truncate(pending, 60))
			}
			pending = strings.TrimSpace(strings.TrimPrefix(trimmed, "- Decision:"))
		case strings.HasPrefix(trimmed, "- Alternative rejected:"):
			pending = ""
		}
	}
	if pending != "" {
		add("decision-has-alternative", SeverityError, design.Path, decisions.Line+len(decisions.Body),
			"the decision %q records no rejected alternative", truncate(pending, 60))
	}
}

func checkReview(change *Change, add adder) {
	review := change.Artifacts["review"]
	if review == nil {
		add("review-exists", SeverityError, "review.md", 0,
			"the change has no review.md. Findings recorded as prose inside the design "+
				"cannot be read as a set, which is why they live in their own artifact")
		return
	}
	for _, heading := range []string{"## Rubric", "## Recommendation", "## Owner decision"} {
		if review.Section(heading) == nil {
			add("review-exists", SeverityError, review.Path, 0,
				"review.md has no %q section", heading)
		}
	}
	decision := review.Section("## Owner decision")
	if decision == nil {
		return
	}
	body := strings.ToLower(strings.Join(decision.Body, " "))
	switch {
	case strings.Contains(body, "pending"), strings.TrimSpace(body) == "":
		add("review-decision-current", SeverityError, review.Path, decision.Line,
			"the owner decision is still pending. Model critique is evidence, never approval, "+
				"so a phase whose review is pending is not done")
	}
}

func checkPhases(change *Change, add adder) {
	tasks := change.Artifacts["tasks"]
	if tasks == nil {
		add("phase-shape-declared", SeverityError, "tasks.md", 0, "the change has no tasks.md")
		return
	}
	for _, phase := range change.Phases {
		if rolloutRE.MatchString(phase.Name) {
			continue // A rollout phase sequences impactful actions and declares no shape.
		}
		shape := phase.Markers["shape"]
		switch shape {
		case "loop", "graph":
		case "":
			add("phase-shape-declared", SeverityError, tasks.Path, phase.Line,
				"phase %d %q declares no SHAPE. Use loop or graph", phase.Number, phase.Name)
			continue
		default:
			add("phase-shape-declared", SeverityError, tasks.Path, phase.Line,
				"phase %d %q declares SHAPE %q. Only loop and graph are valid",
				phase.Number, phase.Name, shape)
			continue
		}

		if shape == "loop" {
			checkLoopPhase(phase, tasks.Path, add)
		}
		if shape == "graph" && len(phase.Tasks) > 1 && phase.Markers["merge"] == "" {
			add("graph-merge-declared", SeverityError, tasks.Path, phase.Line,
				"graph phase %d %q declares no MERGE. Without it the merge is read out of "+
					"file order, which is what the declared graph exists to stop", phase.Number, phase.Name)
		}

		hasReview := false
		for _, task := range phase.Tasks {
			if reviewRE.MatchString(task.Text) {
				hasReview = true
				break
			}
		}
		if !hasReview {
			add("phase-has-review", SeverityError, tasks.Path, phase.Line,
				"phase %d %q has no adversarial-review task", phase.Number, phase.Name)
		}
	}
}

func checkLoopPhase(phase Phase, path string, add adder) {
	// A loop needs both bounds because they fail differently. STOP is the
	// success exit. MAX-ITERS caps the iterations. TERMINAL names how the loop
	// ends without success. MAX-ITERS alone is not enough: a loop that stalls
	// burns the whole cap before anyone notices.
	for _, marker := range []string{"stop", "max-iters", "terminal"} {
		if phase.Markers[marker] == "" {
			add("loop-markers", SeverityError, path, phase.Line,
				"loop phase %d %q declares no %s", phase.Number, phase.Name, strings.ToUpper(marker))
		}
	}
	if stop := phase.Markers["stop"]; stop != "" && !stopCommandRE.MatchString(stop) {
		add("stop-is-a-command", SeverityError, path, phase.Line,
			"the STOP for phase %d %q names no command, so nothing can evaluate it and the "+
				"loop ends when the model decides it is done. Write a command in backticks",
			phase.Number, phase.Name)
	}
	if iters := phase.Markers["max-iters"]; iters != "" {
		if _, err := strconv.Atoi(strings.Fields(iters)[0]); err != nil {
			add("loop-markers", SeverityError, path, phase.Line,
				"the MAX-ITERS for phase %d %q is %q, which is not a number",
				phase.Number, phase.Name, iters)
		}
	}
}

func checkTaskGraph(change *Change, add adder) {
	tasks := change.Artifacts["tasks"]
	if tasks == nil {
		return
	}
	for _, phase := range change.Phases {
		ids := map[string]bool{}
		for _, task := range phase.Tasks {
			if task.ID == "" {
				add("task-id-required", SeverityError, tasks.Path, task.Line,
					"the task %q carries no N.M identifier", truncate(task.Text, 60))
				continue
			}
			ids[task.ID] = true
			if prefix, _, _ := strings.Cut(task.ID, "."); prefix != strconv.Itoa(phase.Number) {
				add("task-id-matches-phase", SeverityError, tasks.Path, task.Line,
					"task %s sits in phase %d", task.ID, phase.Number)
			}
		}
		for _, task := range phase.Tasks {
			for _, dep := range task.Deps {
				if !ids[dep] {
					add("task-deps-resolve", SeverityError, tasks.Path, task.Line,
						"task %s depends on %s, which is not a task in phase %d. A cross-phase "+
							"edge means the phase boundary is in the wrong place",
						task.ID, dep, phase.Number)
				}
			}
		}
		if cycle := findCycle(phase.Tasks); len(cycle) > 0 {
			add("task-deps-acyclic", SeverityError, tasks.Path, phase.Line,
				"phase %d has a dependency cycle: %s", phase.Number, strings.Join(cycle, " -> "))
		}
	}
}

// findCycle returns one cycle in the phase's declared dependencies, or nil.
func findCycle(tasks []Task) []string {
	deps := map[string][]string{}
	for _, task := range tasks {
		if task.ID != "" {
			deps[task.ID] = task.Deps
		}
	}
	const (
		unvisited = 0
		active    = 1
		done      = 2
	)
	state := map[string]int{}
	var path, cycle []string

	var walk func(id string) bool
	walk = func(id string) bool {
		state[id] = active
		path = append(path, id)
		for _, dep := range deps[id] {
			if _, known := deps[dep]; !known {
				continue // task-deps-resolve reports this one.
			}
			switch state[dep] {
			case active:
				for i, seen := range path {
					if seen == dep {
						cycle = append(append([]string{}, path[i:]...), dep)
						return true
					}
				}
				return true
			case unvisited:
				if walk(dep) {
					return true
				}
			}
		}
		path = path[:len(path)-1]
		state[id] = done
		return false
	}

	ids := make([]string, 0, len(deps))
	for id := range deps {
		ids = append(ids, id)
	}
	sort.Strings(ids) // Deterministic order, so the reported cycle is stable.
	for _, id := range ids {
		if state[id] == unvisited && walk(id) {
			return cycle
		}
	}
	return nil
}

func checkSpecs(change *Change, add adder) {
	for _, dir := range specDirs(change) {
		path := filepath.Join(dir, "spec.md")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, req := range ParseRequirements(string(data)) {
			negatives := 0
			for _, scenario := range req.Scenarios {
				switch scenario.Polarity {
				case "negative":
					negatives++
				case "positive":
				default:
					add("scenario-polarity", SeverityError, path, scenario.Line,
						"the scenario %q declares no POLARITY. The lint reads the declared "+
							"marker rather than guessing from prose, so the verdict does not "+
							"depend on wording", truncate(scenario.Name, 50))
				}
			}
			if negatives == 0 {
				add("spec-negative-scenario", SeverityError, path, req.Line,
					"the requirement %q has no declared-negative scenario. A requirement "+
						"nothing can fail is not testable", truncate(req.Name, 50))
			}
		}
	}
}

// specDirs returns each capability directory under the change's specs/.
func specDirs(change *Change) []string {
	entries, err := os.ReadDir(filepath.Join(change.Dir, "specs"))
	if err != nil {
		return nil
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(change.Dir, "specs", entry.Name()))
		}
	}
	return dirs
}

func checkCitations(change *Change, add adder) {
	lock := filepath.Join(change.Dir, "citations.lock")
	if _, err := os.Stat(lock); err != nil {
		add("citations-lock-present", SeverityError, lock, 0,
			"the change carries no citations.lock. An absent lock cannot distinguish "+
				"'there was nothing to cite' from 'nobody looked'. Write {\"records\":[]} to say the first")
		return
	}

	recorded, err := recordIDs(change.Dir)
	if err != nil {
		add("citations-lock-present", SeverityError, lock, 0, "could not read citations.lock: %v", err)
		return
	}
	cited := map[string]bool{}
	for _, artifact := range change.Artifacts {
		for _, m := range citeRE.FindAllStringSubmatch(artifact.Text, -1) {
			id := strings.TrimSpace(m[1])
			cited[id] = true
			if !contains(recorded, id) {
				add("citation-dangling", SeverityError, artifact.Path, 0,
					"[cite: %s] names no record in citations.lock. A dangling reference is a "+
						"claim pretending to be pinned, which is worse than an unpinned one", id)
			}
		}
	}
	for _, id := range recorded {
		if !cited[id] {
			add("citation-uncited", SeverityError, lock, 0,
				"the record %q is cited nowhere. Either the claim was edited out, in which "+
					"case delete the record, or the prose forgot to anchor it with [cite: %s]", id, id)
		}
	}
}

func checkProse(change *Change, add adder) {
	for _, name := range []string{"proposal", "design", "tasks", "review"} {
		artifact := change.Artifacts[name]
		if artifact == nil {
			continue
		}
		inFence := false
		for i, line := range strings.Split(artifact.Text, "\n") {
			if fenceRE.MatchString(line) {
				inFence = !inFence
				continue
			}
			if inFence {
				continue
			}
			if strings.Contains(line, "—") {
				add("no-em-dash", SeverityError, artifact.Path, i+1,
					"the line uses an em-dash. Write so, because, or a full stop instead")
			}
			if m := boldLeadRE.FindStringSubmatch(line); m != nil {
				label := strings.TrimRight(strings.TrimSpace(m[1]), ":.")
				if !allowedBoldLeads[strings.ToUpper(label)] {
					add("bolded-bullet-lead", SeverityError, artifact.Path, i+1,
						"the list item opens with the bolded label %q. Delete the label and "+
							"write the sentence, or carry the force with MUST or SHOULD", label)
				}
			}
		}
	}
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func contains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
