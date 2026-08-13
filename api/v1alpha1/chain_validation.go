package v1alpha1

import (
	"errors"
	"fmt"
)

// Sentinel errors, so a caller can match on what went wrong rather than
// on a message.
var (
	// errFailed reports an operation that did not complete.
	errFailed = errors.New("failed")
)

// ValidateChainDAG checks that the steps form a valid DAG:
//   - no cycles
//   - no undefined step references in DependsOn, ContextFrom, or BranchFrom
func ValidateChainDAG(steps []ChainStep) error {
	// Build name set
	names := make(map[string]struct{}, len(steps))
	for _, s := range steps {
		if _, exists := names[s.Name]; exists {
			return fmt.Errorf("%w: duplicate step name %q", errFailed, s.Name)
		}
		names[s.Name] = struct{}{}
	}

	// Validate references
	for _, s := range steps {
		for _, dep := range s.DependsOn {
			if _, ok := names[dep]; !ok {
				return fmt.Errorf("%w: step %q depends on undefined step %q", errFailed, s.Name, dep)
			}
		}
		if s.ContextFrom != "" {
			if _, ok := names[s.ContextFrom]; !ok {
				return fmt.Errorf("%w: step %q contextFrom references undefined step %q", errFailed, s.Name, s.ContextFrom)
			}
		}
		if s.BranchFrom != "" {
			if _, ok := names[s.BranchFrom]; !ok {
				return fmt.Errorf("%w: step %q branchFrom references undefined step %q", errFailed, s.Name, s.BranchFrom)
			}
		}
	}

	// Kahn's algorithm for cycle detection
	inDegree := make(map[string]int, len(steps))
	adj := make(map[string][]string, len(steps))
	for _, s := range steps {
		if _, ok := inDegree[s.Name]; !ok {
			inDegree[s.Name] = 0
		}
		for _, dep := range s.DependsOn {
			adj[dep] = append(adj[dep], s.Name)
			inDegree[s.Name]++
		}
	}

	queue := []string{}
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	visited := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visited++
		for _, neighbor := range adj[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if visited != len(steps) {
		return fmt.Errorf("%w: chain contains a cycle", errFailed)
	}
	return nil
}
