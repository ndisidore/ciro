// Package pipeline defines the core types for a ciro CI/CD pipeline.
package pipeline

import (
	"errors"
	"fmt"
)

// Sentinel errors for pipeline validation.
var (
	ErrEmptyPipeline  = errors.New("pipeline has no steps")
	ErrEmptyStepName  = errors.New("step has empty name")
	ErrDuplicateStep  = errors.New("duplicate step name")
	ErrMissingImage   = errors.New("step missing image")
	ErrMissingRun     = errors.New("step has no run commands")
	ErrSelfDependency = errors.New("step depends on itself")
	ErrUnknownDep     = errors.New("unknown dependency")
	ErrCycleDetected  = errors.New("dependency cycle detected")
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

		if s.Name == "" {
			return fmt.Errorf("step at index %d: %w", i, ErrEmptyStepName)
		}

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

	if err := checkSelfDeps(p.Steps); err != nil {
		return err
	}

	// topoSort validates unknown deps and cycles via 3-state DFS.
	_, err := p.TopoSort()
	return err
}

// checkSelfDeps detects steps that list themselves as a dependency.
func checkSelfDeps(steps []Step) error {
	for i := range steps {
		for _, dep := range steps[i].DependsOn {
			if dep == steps[i].Name {
				return fmt.Errorf(
					"step %q depends on itself: %w", steps[i].Name, ErrSelfDependency,
				)
			}
		}
	}
	return nil
}

// TopoSort returns step indices in topological order (dependencies first).
// Returns an error if an unknown dependency is encountered.
func (p *Pipeline) TopoSort() ([]int, error) {
	g := newStepGraph(p.Steps)
	return g.topoSort()
}

// stepGraph provides indexed graph operations over a step slice.
type stepGraph struct {
	steps []Step
	index map[string]int
}

func newStepGraph(steps []Step) stepGraph {
	idx := make(map[string]int, len(steps))
	for i := range steps {
		idx[steps[i].Name] = i
	}
	return stepGraph{steps: steps, index: idx}
}

// resolveDep looks up a dependency by name, returning a clear error for unknown deps.
func (g *stepGraph) resolveDep(stepName, dep string) (int, error) {
	j, ok := g.index[dep]
	if !ok {
		return 0, fmt.Errorf(
			"step %q depends on %q: %w", stepName, dep, ErrUnknownDep,
		)
	}
	return j, nil
}

func (g *stepGraph) topoSort() ([]int, error) {
	const (
		unvisited = iota
		visiting
		visited
	)

	state := make([]int, len(g.steps))
	order := make([]int, 0, len(g.steps))

	var visit func(int) error
	visit = func(i int) error {
		switch state[i] {
		case visited:
			return nil
		case visiting:
			return fmt.Errorf("step %q: %w", g.steps[i].Name, ErrCycleDetected)
		}
		state[i] = visiting
		for _, dep := range g.steps[i].DependsOn {
			j, err := g.resolveDep(g.steps[i].Name, dep)
			if err != nil {
				return err
			}
			if err := visit(j); err != nil {
				return err
			}
		}
		state[i] = visited
		order = append(order, i)
		return nil
	}

	for i := range g.steps {
		if err := visit(i); err != nil {
			return nil, err
		}
	}
	return order, nil
}
