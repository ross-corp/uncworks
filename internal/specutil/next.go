package specutil

import (
	"fmt"
	"strings"

	"github.com/uncworks/aot/internal/citelock"
)

// recordIDs returns the citation record ids pinned by a change.
func recordIDs(dir string) ([]string, error) {
	ids, err := citelock.RecordIDs(dir)
	if err != nil {
		return nil, fmt.Errorf("reading citation records: %w", err)
	}
	return ids, nil
}

// Ready is the answer to "what can I work on now".
type Ready struct {
	Change string `json:"change"`
	// Phase is the first phase with unfinished work.
	Phase   int       `json:"phase"`
	Name    string    `json:"phaseName"`
	Shape   string    `json:"shape"`
	Stop    string    `json:"stop,omitempty"`
	Merge   string    `json:"merge,omitempty"`
	Tasks   []Task    `json:"ready"`
	Blocked []Blocked `json:"blocked,omitempty"`
	// Concurrent is true when more than one ready task can run at once. A loop
	// phase is never concurrent, because its next iteration reads what the
	// current one wrote.
	Concurrent bool `json:"runnableConcurrently"`
	// Done is true when every task in the change is checked off.
	Done bool `json:"done"`
}

// Blocked is one task that cannot start yet, with the reason.
type Blocked struct {
	ID      string   `json:"id"`
	Waiting []string `json:"waitingOn"`
}

// Next reports the runnable set for a change.
//
// Read the ready set from here rather than from the top of tasks.md. The file
// declares a dependency graph and a phase shape, and reading in file order
// ignores both, which makes the declaration decoration.
func Next(change *Change) (Ready, error) {
	ready := Ready{Change: change.Name, Done: true}

	for _, phase := range change.Phases {
		open := false
		for _, task := range phase.Tasks {
			if !task.Done {
				open = true
				break
			}
		}
		if !open {
			continue
		}
		ready.Done = false
		ready.Phase = phase.Number
		ready.Name = phase.Name
		ready.Shape = phase.Markers["shape"]
		ready.Stop = phase.Markers["stop"]
		ready.Merge = phase.Markers["merge"]

		done := map[string]bool{}
		known := map[string]bool{}
		for _, task := range phase.Tasks {
			if task.ID != "" {
				known[task.ID] = true
				done[task.ID] = task.Done
			}
		}
		if cycle := findCycle(phase.Tasks); len(cycle) > 0 {
			return ready, fmt.Errorf("%w in phase %d: %s",
				ErrCycle, phase.Number, strings.Join(cycle, " -> "))
		}

		for _, task := range phase.Tasks {
			if task.Done {
				continue
			}
			var waiting []string
			for _, dep := range task.Deps {
				if known[dep] && !done[dep] {
					waiting = append(waiting, dep)
				}
			}
			if len(waiting) == 0 {
				ready.Tasks = append(ready.Tasks, task)
			} else {
				ready.Blocked = append(ready.Blocked, Blocked{ID: task.ID, Waiting: waiting})
			}
		}

		// Concurrency is only claimed for a graph phase, and only when no ready
		// task is an owner gate or a review. Neither may be delegated.
		if ready.Shape == "graph" && len(ready.Tasks) > 1 {
			ready.Concurrent = true
			for _, task := range ready.Tasks {
				if isGate(task.Text) || reviewRE.MatchString(task.Text) {
					ready.Concurrent = false
					break
				}
			}
		}
		return ready, nil
	}
	return ready, nil
}

// isGate reports whether a task is an owner gate. An `Apply:` task performs an
// impactful action and a `Confirm:` task states a judgment only the owner can
// make, so neither is the agent's to check off.
func isGate(text string) bool {
	lowered := strings.ToLower(strings.TrimSpace(text))
	return strings.HasPrefix(lowered, "apply:") || strings.HasPrefix(lowered, "confirm:")
}

// WriteConflicts returns the ready tasks whose declared write sets intersect.
//
// A ready set never holds two tasks with an edge between them, so a shared file
// is the only coupling the scheduler cannot see. Two tasks that write one file
// produce a merge conflict rather than parallel progress.
func WriteConflicts(ready Ready) map[string][]string {
	owners := map[string][]string{}
	for _, task := range ready.Tasks {
		for _, path := range task.Writes {
			owners[path] = append(owners[path], task.ID)
		}
	}
	conflicts := map[string][]string{}
	for path, ids := range owners {
		if len(ids) > 1 {
			conflicts[path] = ids
		}
	}
	return conflicts
}

// Progress counts the tasks in a change.
type Progress struct {
	Change string `json:"change"`
	Done   int    `json:"done"`
	Total  int    `json:"total"`
}

// Status counts completed tasks across a change.
func Status(change *Change) Progress {
	progress := Progress{Change: change.Name}
	for _, task := range change.AllTasks() {
		progress.Total++
		if task.Done {
			progress.Done++
		}
	}
	return progress
}

// Mermaid renders the phases of a change as a Mermaid flowchart, so the shape
// a reader is asked to trust is the shape the tool parsed.
func Mermaid(change *Change) string {
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	for _, phase := range change.Phases {
		shape := phase.Markers["shape"]
		if shape == "" {
			shape = "rollout"
		}
		fmt.Fprintf(&b, "    subgraph p%d[\"%d. %s (%s)\"]\n", phase.Number, phase.Number, phase.Name, shape)
		for _, task := range phase.Tasks {
			mark := " "
			if task.Done {
				mark = "x"
			}
			fmt.Fprintf(&b, "        t%s[\"[%s] %s %s\"]\n",
				sanitize(task.ID), mark, task.ID, sanitize(truncate(task.Text, 48)))
		}
		b.WriteString("    end\n")
		for _, task := range phase.Tasks {
			for _, dep := range task.Deps {
				fmt.Fprintf(&b, "    t%s --> t%s\n", sanitize(dep), sanitize(task.ID))
			}
		}
	}
	for i := 1; i < len(change.Phases); i++ {
		fmt.Fprintf(&b, "    p%d --> p%d\n", change.Phases[i-1].Number, change.Phases[i].Number)
	}
	return b.String()
}

// sanitize strips the characters Mermaid treats as syntax.
func sanitize(s string) string {
	return strings.NewReplacer(
		".", "_", "\"", "'", "[", "(", "]", ")", "{", "(", "}", ")", "|", "/",
	).Replace(s)
}
