// Package specutil reads an OpenSpec change directory and answers two
// questions about it deterministically: does it satisfy the rubric, and what is
// runnable now.
//
// Both answers are pure functions of the files on disk. Nothing here reaches
// the network or an LLM, so two runs over the same tree always agree. That is
// what lets the results gate a commit.
package specutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Change is one parsed change directory.
type Change struct {
	Name      string
	Dir       string
	Artifacts map[string]*Artifact
	Phases    []Phase
}

// Artifact is one markdown file inside a change.
type Artifact struct {
	// ID is the artifact name without its extension: proposal, design, tasks,
	// or review.
	ID       string
	Path     string
	Text     string
	Sections []Section
}

// Section is one markdown heading and the lines under it, up to the next
// heading at the same level or higher.
type Section struct {
	Heading string
	Level   int
	Line    int
	Body    []string
}

// Phase is one `## <n>. Name` heading in tasks.md, with its markers and tasks.
type Phase struct {
	Number  int
	Name    string
	Line    int
	Markers map[string]string
	Tasks   []Task
}

// Task is one checkbox in tasks.md.
type Task struct {
	// ID is the `N.M` identifier that opens the task text.
	ID       string
	Text     string
	Done     bool
	Line     int
	Deps     []string
	Writes   []string
	Artifact string
}

// Requirement is one `### Requirement:` heading in a spec file, with the
// scenarios under it.
type Requirement struct {
	Name      string
	Line      int
	Scenarios []Scenario
}

// Scenario is one `#### Scenario:` heading under a requirement.
type Scenario struct {
	Name string
	Line int
	// Polarity is the value of the `- **POLARITY**` body line, lowercased. It
	// is empty when the scenario declares none.
	//
	// The marker lives in the body rather than in the heading because
	// openspec's archive step matches `#### Scenario:` strictly, and a heading
	// like `#### Scenario [negative]:` makes archive drop the scenario without
	// saying so.
	Polarity string
}

var (
	headingRE = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	phaseRE   = regexp.MustCompile(`^##\s+(\d+)\.\s*(.*)$`)
	taskRE    = regexp.MustCompile(`^\s*-\s+\[([ xX])\]\s+(.*)$`)
	taskIDRE  = regexp.MustCompile(`^(\d+\.\d+)\s+(.*)$`)
	markerRE  = regexp.MustCompile(`^\s*-\s+\*\*([A-Z][A-Z-]*)\*\*\s*(.*)$`)
	// The value runs to the next backtick, so a line may carry both fields and
	// neither swallows the other.
	fieldRE = regexp.MustCompile("`(deps|writes):`\\s*([^`]*)")
	fenceRE = regexp.MustCompile("^\\s*```")
)

// artifactNames are the files a change may carry, in the order the schema
// generates them.
var artifactNames = []string{"proposal", "citations", "design", "tasks", "review"}

// Load reads one change directory.
func Load(dir string) (*Change, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("reading change directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s is not a directory", ErrNotAChange, dir)
	}

	change := &Change{
		Name:      filepath.Base(dir),
		Dir:       dir,
		Artifacts: map[string]*Artifact{},
	}
	for _, name := range artifactNames {
		if name == "citations" {
			continue // The lockfile is JSON, and citelock owns it.
		}
		path := filepath.Join(dir, name+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			continue // An absent artifact is a rubric finding, not a parse error.
		}
		text := string(data)
		change.Artifacts[name] = &Artifact{
			ID:       name,
			Path:     path,
			Text:     text,
			Sections: parseSections(text),
		}
	}
	if tasks, ok := change.Artifacts["tasks"]; ok {
		change.Phases = parsePhases(tasks.Text)
	}
	return change, nil
}

// LoadAll reads every change directory under changesDir, skipping `archive`.
func LoadAll(changesDir string) ([]*Change, error) {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return nil, fmt.Errorf("reading changes directory: %w", err)
	}
	var changes []*Change
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "archive" {
			continue
		}
		change, err := Load(filepath.Join(changesDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}

// parseSections splits markdown into headings and their bodies. Fenced code
// blocks are opaque, so a `#` inside one is not a heading.
func parseSections(text string) []Section {
	var sections []Section
	var current *Section
	inFence := false

	for i, line := range strings.Split(text, "\n") {
		if fenceRE.MatchString(line) {
			inFence = !inFence
		}
		if !inFence {
			if m := headingRE.FindStringSubmatch(line); m != nil {
				sections = append(sections, Section{
					Heading: strings.TrimSpace(line),
					Level:   len(m[1]),
					Line:    i + 1,
				})
				current = &sections[len(sections)-1]
				continue
			}
		}
		if current != nil {
			current.Body = append(current.Body, line)
		}
	}
	return sections
}

// parsePhases reads the `## <n>. Name` phases out of tasks.md, with the markers
// and checkboxes that belong to each.
func parsePhases(text string) []Phase {
	var phases []Phase
	var current *Phase
	inFence := false

	for i, line := range strings.Split(text, "\n") {
		if fenceRE.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		if m := phaseRE.FindStringSubmatch(line); m != nil {
			number, _ := strconv.Atoi(m[1])
			phases = append(phases, Phase{
				Number:  number,
				Name:    strings.TrimSpace(m[2]),
				Line:    i + 1,
				Markers: map[string]string{},
			})
			current = &phases[len(phases)-1]
			continue
		}
		if current == nil {
			continue
		}

		if m := taskRE.FindStringSubmatch(line); m != nil {
			current.Tasks = append(current.Tasks, parseTask(m[2], m[1] != " ", i+1))
			continue
		}
		// A marker only counts outside a checkbox, so `- **STOP** ...` is a
		// phase marker and a bolded word inside a task is not.
		if m := markerRE.FindStringSubmatch(line); m != nil {
			current.Markers[strings.ToLower(m[1])] = strings.TrimSpace(m[2])
		}
	}
	return phases
}

func parseTask(text string, done bool, line int) Task {
	task := Task{Text: strings.TrimSpace(text), Done: done, Line: line}

	// The field pass takes the first match on the line. A task whose prose
	// repeats a marker label would otherwise capture the prose as the value.
	for _, m := range fieldRE.FindAllStringSubmatch(task.Text, -1) {
		values := splitList(m[2])
		switch m[1] {
		case "deps":
			if task.Deps == nil {
				task.Deps = values
			}
		case "writes":
			if task.Writes == nil {
				task.Writes = values
			}
		}
	}
	task.Text = strings.TrimSpace(fieldRE.ReplaceAllString(task.Text, ""))

	if m := taskIDRE.FindStringSubmatch(task.Text); m != nil {
		task.ID = m[1]
		task.Text = strings.TrimSpace(m[2])
	}
	return task
}

// splitList reads a comma or space separated marker value. The literal `none`
// means an empty list, which is how a task states it has no dependency rather
// than leaving the question open.
func splitList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "none") {
		return []string{}
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if field = strings.Trim(field, "`"); field != "" {
			out = append(out, field)
		}
	}
	return out
}

// ParseRequirements reads the requirements and scenarios out of a spec file.
func ParseRequirements(text string) []Requirement {
	var reqs []Requirement
	var scenario *Scenario
	inFence := false

	for i, line := range strings.Split(text, "\n") {
		if fenceRE.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "### Requirement:"):
			reqs = append(reqs, Requirement{
				Name: strings.TrimSpace(strings.TrimPrefix(trimmed, "### Requirement:")),
				Line: i + 1,
			})
			scenario = nil
		case strings.HasPrefix(trimmed, "#### Scenario:") && len(reqs) > 0:
			req := &reqs[len(reqs)-1]
			req.Scenarios = append(req.Scenarios, Scenario{
				Name: strings.TrimSpace(strings.TrimPrefix(trimmed, "#### Scenario:")),
				Line: i + 1,
			})
			scenario = &req.Scenarios[len(req.Scenarios)-1]
		case scenario != nil:
			if m := markerRE.FindStringSubmatch(line); m != nil && m[1] == "POLARITY" {
				scenario.Polarity = strings.ToLower(strings.TrimSpace(m[2]))
			}
		}
	}
	return reqs
}

// Section returns the named section of an artifact.
func (a *Artifact) Section(heading string) *Section {
	if a == nil {
		return nil
	}
	for i := range a.Sections {
		if strings.EqualFold(a.Sections[i].Heading, heading) {
			return &a.Sections[i]
		}
	}
	return nil
}

// Bullets returns the top-level list items in a section.
func (s *Section) Bullets() []string {
	if s == nil {
		return nil
	}
	var bullets []string
	inFence := false
	for _, line := range s.Body {
		if fenceRE.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		trimmed := strings.TrimLeft(line, " \t")
		indent := len(line) - len(trimmed)
		if indent > 1 {
			continue // A nested item belongs to its parent, not to the section.
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			bullets = append(bullets, strings.TrimSpace(trimmed[2:]))
		}
	}
	return bullets
}

// AllTasks returns every task in the change, in file order.
func (c *Change) AllTasks() []Task {
	var tasks []Task
	for _, phase := range c.Phases {
		tasks = append(tasks, phase.Tasks...)
	}
	return tasks
}
