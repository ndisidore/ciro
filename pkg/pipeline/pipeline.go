// Package pipeline defines the core types for a ciro CI/CD pipeline.
package pipeline

import (
	"errors"
	"fmt"
)

// Sentinel errors for pipeline validation.
var (
	ErrEmptyPipeline = errors.New("pipeline has no steps")
	ErrDuplicateStep = errors.New("duplicate step name")
	ErrMissingImage  = errors.New("step missing image")
	ErrMissingRun    = errors.New("step has no run commands")
	ErrUnknownDep    = errors.New("unknown dependency")
	ErrCycleDetected = errors.New("dependency cycle detected")
)

// Pipeline represents a CI/CD pipeline parsed from KDL.
type Pipeline struct {
	Name  string
	Steps []Step
}

// Step represents a single execution unit within a pipeline.
type Step struct {
	Name      string
	Image     string
	Run       []string
	Workdir   string
	DependsOn []string
	Mounts    []Mount
	Caches    []Cache
}

// Mount represents a bind mount from host to container.
type Mount struct {
	Source string
	Target string
}

// Cache represents a persistent cache volume.
type Cache struct {
	ID     string
	Target string
}

// Validate checks that the pipeline is well-formed and returns the first error found.
func (p *Pipeline) Validate() error {
	if len(p.Steps) == 0 {
		return ErrEmptyPipeline
	}

	names := make(map[string]struct{}, len(p.Steps))
	for i := range p.Steps {
		s := &p.Steps[i]

		if _, exists := names[s.Name]; exists {
			return fmt.Errorf("step %q: %w", s.Name, ErrDuplicateStep)
		}
		names[s.Name] = struct{}{}

		if s.Image == "" {
			return fmt.Errorf("step %q: %w", s.Name, ErrMissingImage)
		}
		if len(s.Run) == 0 {
			return fmt.Errorf("step %q: %w", s.Name, ErrMissingRun)
		}
	}

	if err := checkDepRefs(p.Steps, names); err != nil {
		return err
	}
	return detectCycles(p.Steps)
}

// checkDepRefs verifies that all dependency references point to existing steps.
func checkDepRefs(steps []Step, names map[string]struct{}) error {
	for i := range steps {
		for _, dep := range steps[i].DependsOn {
			if _, exists := names[dep]; !exists {
				return fmt.Errorf(
					"step %q depends on %q: %w", steps[i].Name, dep, ErrUnknownDep,
				)
			}
		}
	}
	return nil
}

// detectCycles uses DFS coloring to find dependency cycles among steps.
func detectCycles(steps []Step) error {
	const (
		unvisited = iota
		visiting
		visited
	)

	index := make(map[string]int, len(steps))
	for i := range steps {
		index[steps[i].Name] = i
	}

	state := make([]int, len(steps))

	var visit func(int) error
	visit = func(i int) error {
		switch state[i] {
		case visited:
			return nil
		case visiting:
			return fmt.Errorf("step %q: %w", steps[i].Name, ErrCycleDetected)
		}
		state[i] = visiting
		for _, dep := range steps[i].DependsOn {
			if err := visit(index[dep]); err != nil {
				return err
			}
		}
		state[i] = visited
		return nil
	}

	for i := range steps {
		if err := visit(i); err != nil {
			return err
		}
	}
	return nil
}

// TopoSort returns step indices in topological order (dependencies first).
// It assumes the pipeline has already been validated (no cycles or missing deps).
func (p *Pipeline) TopoSort() []int {
	index := make(map[string]int, len(p.Steps))
	for i := range p.Steps {
		index[p.Steps[i].Name] = i
	}

	visited := make([]bool, len(p.Steps))
	order := make([]int, 0, len(p.Steps))

	var visit func(int)
	visit = func(i int) {
		if visited[i] {
			return
		}
		visited[i] = true
		for _, dep := range p.Steps[i].DependsOn {
			visit(index[dep])
		}
		order = append(order, i)
	}

	for i := range p.Steps {
		visit(i)
	}
	return order
}
