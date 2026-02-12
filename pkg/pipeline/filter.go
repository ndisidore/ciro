package pipeline

import (
	"errors"
	"fmt"
)

// Sentinel errors for step filtering.
var (
	ErrUnknownStartAt    = errors.New("start-at step not found")
	ErrUnknownStopAfter  = errors.New("stop-after step not found")
	ErrEmptyFilterResult = errors.New("filter produces no steps")
)

// FilterOpts configures partial pipeline execution.
type FilterOpts struct {
	StartAt   string // run from this step forward (includes downstream dependents)
	StopAfter string // run up to and including this step (excludes downstream)
}

// stepAdj holds forward (dependents) and backward (dependencies) adjacency
// maps pre-computed from a step slice.
type stepAdj struct {
	dependents   map[string][]string
	dependencies map[string][]string
}

// buildStepAdj builds adjacency maps and validates that StartAt / StopAfter
// refer to existing steps.
func buildStepAdj(steps []Step, opts FilterOpts) (stepAdj, error) {
	nameSet := make(map[string]struct{}, len(steps))
	adj := stepAdj{
		dependents:   make(map[string][]string, len(steps)),
		dependencies: make(map[string][]string, len(steps)),
	}
	for i := range steps {
		name := steps[i].Name
		nameSet[name] = struct{}{}
		adj.dependencies[name] = steps[i].DependsOn
		for _, dep := range steps[i].DependsOn {
			adj.dependents[dep] = append(adj.dependents[dep], name)
		}
	}
	if opts.StartAt != "" {
		if _, ok := nameSet[opts.StartAt]; !ok {
			return stepAdj{}, fmt.Errorf("%q: %w", opts.StartAt, ErrUnknownStartAt)
		}
	}
	if opts.StopAfter != "" {
		if _, ok := nameSet[opts.StopAfter]; !ok {
			return stepAdj{}, fmt.Errorf("%q: %w", opts.StopAfter, ErrUnknownStopAfter)
		}
	}
	return adj, nil
}

// FilterSteps selects a subgraph of steps based on start-at / stop-after
// options. Steps in the execution window retain their exports; transitive
// dependencies included only for build correctness have exports stripped.
// The original slice order is preserved.
func FilterSteps(steps []Step, opts FilterOpts) ([]Step, error) {
	if opts.StartAt == "" && opts.StopAfter == "" {
		return steps, nil
	}

	adj, err := buildStepAdj(steps, opts)
	if err != nil {
		return nil, err
	}

	// Forward reachable from StartAt (the step itself + all downstream dependents).
	var forwardSet map[string]struct{}
	if opts.StartAt != "" {
		forwardSet = bfsReachable(opts.StartAt, adj.dependents)
	}

	// Backward reachable from StopAfter (the step itself + all upstream dependencies).
	var backwardSet map[string]struct{}
	if opts.StopAfter != "" {
		backwardSet = bfsReachable(opts.StopAfter, adj.dependencies)
	}

	executionWindow := intersectSets(forwardSet, backwardSet)
	if len(executionWindow) == 0 {
		return nil, fmt.Errorf(
			"start-at=%q stop-after=%q: %w",
			opts.StartAt, opts.StopAfter, ErrEmptyFilterResult,
		)
	}

	// Add transitive dependencies of execution window members.
	fullSet := make(map[string]struct{}, len(executionWindow))
	for name := range executionWindow {
		fullSet[name] = struct{}{}
	}
	for name := range executionWindow {
		addTransitiveDeps(name, adj.dependencies, fullSet)
	}

	// Filter steps preserving original order; strip exports from dep-only steps.
	filtered := make([]Step, 0, len(fullSet))
	for i := range steps {
		if _, ok := fullSet[steps[i].Name]; !ok {
			continue
		}
		s := steps[i]
		if _, inWindow := executionWindow[s.Name]; !inWindow {
			s.Exports = nil
		}
		filtered = append(filtered, s)
	}

	return filtered, nil
}

// bfsReachable returns all nodes reachable from start (inclusive) via the
// given adjacency map.
func bfsReachable(start string, adj map[string][]string) map[string]struct{} {
	visited := map[string]struct{}{start: {}}
	queue := []string{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range adj[cur] {
			if _, ok := visited[next]; !ok {
				visited[next] = struct{}{}
				queue = append(queue, next)
			}
		}
	}
	return visited
}

// intersectSets returns the intersection of a and b. A nil set is treated
// as the universe (all elements pass).
func intersectSets(a, b map[string]struct{}) map[string]struct{} {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	// Iterate over the smaller set for efficiency.
	if len(a) > len(b) {
		a, b = b, a
	}
	result := make(map[string]struct{}, len(a))
	for k := range a {
		if _, ok := b[k]; ok {
			result[k] = struct{}{}
		}
	}
	return result
}

// addTransitiveDeps adds all transitive dependencies of name to visited via DFS.
func addTransitiveDeps(name string, deps map[string][]string, visited map[string]struct{}) {
	for _, dep := range deps[name] {
		if _, ok := visited[dep]; ok {
			continue
		}
		visited[dep] = struct{}{}
		addTransitiveDeps(dep, deps, visited)
	}
}
