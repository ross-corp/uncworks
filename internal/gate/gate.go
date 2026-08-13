// Package gate computes the factory-gate verdicts for a pull request.
//
// Every verdict is a pure function of two git refs and the files in them. That
// is the reason this is a command rather than a service: a service would need
// an endpoint, a secret, and an operator to compute what a job can compute from
// the tree it already has.
//
// Two verdicts are required and two are advisory. Tier and citations are
// required, because both are decidable and both are wrong rarely. Conformance
// and order are advisory, because both are wrong often enough that making them
// required would train the owner to override them, which costs more than the
// checks return.
package gate

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

// ErrFailed reports that at least one required verdict failed.
var ErrFailed = errors.New("required gate verdict failed")

// Tier is a change's blast radius.
type Tier string

const (
	// TierAmendment rewrites the durable spec corpus. Code cannot merge against
	// a spec that is under revision, so this tier never passes on its own.
	TierAmendment Tier = "amendment"
	// TierExperiment is a branch with no spec. It may target only another
	// experiment branch.
	TierExperiment Tier = "experiment"
	// TierT1 is confined to documentation and tests, and needs no spec.
	TierT1 Tier = "T1"
	// TierT2 changes one capability's implementation, and needs a merged spec.
	TierT2 Tier = "T2"
	// TierT3 spans capabilities, alters a public surface, or touches
	// deployment. It needs a merged spec and a recorded owner decision.
	TierT3 Tier = "T3"
)

// Verdict is one check's result.
type Verdict struct {
	Name     string   `json:"name"`
	Required bool     `json:"required"`
	Pass     bool     `json:"pass"`
	Summary  string   `json:"summary"`
	Detail   []string `json:"detail,omitempty"`
}

// Result is every verdict for one pull request.
type Result struct {
	Change   string    `json:"change"`
	Tier     Tier      `json:"tier"`
	Reason   string    `json:"tierReason"`
	Verdicts []Verdict `json:"verdicts"`
}

// Only narrows a result to one verdict, so a workflow can publish one check run
// per verdict and give branch protection a name to require.
func (r Result) Only(name string) Result {
	narrowed := Result{Change: r.Change, Tier: r.Tier, Reason: r.Reason}
	for _, v := range r.Verdicts {
		if v.Name == name {
			narrowed.Verdicts = append(narrowed.Verdicts, v)
		}
	}
	return narrowed
}

// Failed reports whether a required verdict failed.
func (r Result) Failed() bool {
	for _, v := range r.Verdicts {
		if v.Required && !v.Pass {
			return true
		}
	}
	return false
}

// Repo is the git access the gate needs. A test supplies a fake, and nothing
// in the verdicts shells out directly.
type Repo interface {
	// ChangedPaths returns the paths that differ between two refs.
	ChangedPaths(ctx context.Context, base, head string) ([]string, error)
	// Exists reports whether a path is present at a ref.
	Exists(ctx context.Context, ref, path string) (bool, error)
}

// Options configures one gate run.
type Options struct {
	// ChangeID names the openspec change this pull request implements. An
	// empty value means the gate derives it from the branch, and a change that
	// needs a spec then fails for the right reason.
	ChangeID string
	Base     string
	Head     string
	// Root is the repository root on disk, used to read the working tree for
	// the change's own artifacts.
	Root string
}

// Check computes every verdict.
func Check(ctx context.Context, repo Repo, opts Options) (Result, error) {
	changed, err := repo.ChangedPaths(ctx, opts.Base, opts.Head)
	if err != nil {
		return Result{}, fmt.Errorf("reading the diff between %s and %s: %w", opts.Base, opts.Head, err)
	}
	sort.Strings(changed)

	tier, reason := ClassifyTier(changed)
	result := Result{Change: opts.ChangeID, Tier: tier, Reason: reason}

	result.Verdicts = append(result.Verdicts,
		tierVerdict(ctx, repo, opts, tier, reason, changed),
		citationsVerdict(opts),
		conformanceVerdict(opts, changed),
		orderVerdict(ctx, repo, opts, changed),
	)
	if result.Failed() {
		return result, ErrFailed
	}
	return result, nil
}

// docPrefixes and docSuffixes mark a path as documentation or test-only.
//
// openspec/ is deliberately absent. A change to openspec/specs/ rewrites the
// contract other work is measured against, so treating it as documentation
// would make a spec amendment the one diff shape the classifier cannot fail.
var (
	docPrefixes  = []string{"docs/", ".github/ISSUE_TEMPLATE/"}
	testSuffixes = []string{"_test.go", ".test.ts", ".spec.ts"}
	testPrefixes = []string{"test/", "e2e/", "web/e2e/"}

	// publicSurfaces are the paths where a change alters something a caller
	// outside this repository can observe.
	//
	// internal/server is here despite the internal/ prefix: it implements the
	// REST and ConnectRPC surface that docs/reference/api.md documents, so a
	// field rename there is observable by every out-of-repo client. The prefix
	// list would otherwise only catch surfaces declared in a schema directory.
	publicSurfaces = []string{
		"proto/", "api/", "deploy/", "docker/", "gen/", "internal/server/",
	}

	// specPrefix marks a change to the durable spec corpus. An open amendment
	// to it blocks code that is measured against the text being rewritten.
	specPrefix = "openspec/specs/"

	// changePrefix marks a change's own artifacts, which are not the diff a
	// conformance or tier check is about.
	changePrefix = "openspec/changes/"
)

// ClassifyTier computes a change's tier from the paths it touches.
//
// The classifier reads only the diff. It does not read the branch name, the
// pull request title, or a label, because all three are writable by the author
// the gate constrains, so a change could otherwise lower its own bar by
// renaming a branch.
func ClassifyTier(changed []string) (Tier, string) {
	if len(changed) == 0 {
		return TierT1, "the diff is empty"
	}

	// A spec amendment is checked before anything else. It is the diff shape
	// the escalation protocol exists for, and reading it as documentation is
	// what let it through.
	var amended []string
	for _, p := range changed {
		if strings.HasPrefix(p, specPrefix) {
			amended = append(amended, p)
		}
	}
	if len(amended) > 0 {
		return TierAmendment, fmt.Sprintf("the diff amends the spec corpus: %s",
			strings.Join(amended, ", "))
	}

	var surfaces []string
	capabilities := map[string]bool{}
	codeChanged := false

	for _, p := range changed {
		switch {
		case isDoc(p), isTest(p), strings.HasPrefix(p, changePrefix):
			continue
		}
		codeChanged = true
		if surface := publicSurface(p); surface != "" {
			surfaces = append(surfaces, surface)
		}
		if capability := capabilityOf(p); capability != "" {
			capabilities[capability] = true
		}
	}

	if !codeChanged {
		return TierT1, "the diff touches only documentation and tests"
	}
	if len(surfaces) > 0 {
		sort.Strings(surfaces)
		return TierT3, fmt.Sprintf("the diff alters a public surface: %s",
			strings.Join(dedupe(surfaces), ", "))
	}
	if len(capabilities) > 1 {
		names := make([]string, 0, len(capabilities))
		for name := range capabilities {
			names = append(names, name)
		}
		sort.Strings(names)
		return TierT3, fmt.Sprintf("the diff spans %d capabilities: %s",
			len(names), strings.Join(names, ", "))
	}
	for name := range capabilities {
		return TierT2, "the diff changes the capability " + name
	}
	return TierT2, "the diff changes implementation code"
}

func isDoc(p string) bool {
	for _, prefix := range docPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return strings.HasSuffix(p, ".md")
}

func isTest(p string) bool {
	for _, prefix := range testPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	for _, suffix := range testSuffixes {
		if strings.HasSuffix(p, suffix) {
			return true
		}
	}
	return false
}

func publicSurface(p string) string {
	for _, prefix := range publicSurfaces {
		if strings.HasPrefix(p, prefix) {
			return strings.TrimSuffix(prefix, "/")
		}
	}
	return ""
}

// capabilityOf names the unit of implementation a path belongs to. For Go code
// that is the package directory, which is the granularity a spec is written at.
func capabilityOf(p string) string {
	switch {
	case strings.HasPrefix(p, "internal/"), strings.HasPrefix(p, "cmd/"):
		parts := strings.Split(p, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	case strings.HasPrefix(p, "web/src/"):
		return "web"
	case strings.HasPrefix(p, "packages/"):
		parts := strings.Split(p, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	return path.Dir(p)
}

func dedupe(items []string) []string {
	out := items[:0:0]
	seen := map[string]bool{}
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}
