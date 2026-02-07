// Package builder converts validated pipelines into BuildKit LLB definitions.
package builder

import (
	"context"
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"

	"github.com/ndisidore/ciro/pkg/pipeline"
)

// Result holds the LLB definitions for a pipeline, one per step in topological order.
type Result struct {
	// Definitions contains one LLB definition per step, ordered by dependency.
	Definitions []*llb.Definition
	// StepNames maps each definition index to its step name.
	StepNames []string
}

// Build converts a pipeline to BuildKit LLB definitions.
// It reuses a cached topological order when available, otherwise validates first.
func Build(ctx context.Context, p pipeline.Pipeline) (Result, error) {
	order := p.TopoOrder
	if !validOrder(order, len(p.Steps)) {
		var err error
		order, err = p.Validate()
		if err != nil {
			return Result{}, fmt.Errorf("validating pipeline: %w", err)
		}
	}

	result := Result{
		Definitions: make([]*llb.Definition, 0, len(order)),
		StepNames:   make([]string, 0, len(order)),
	}

	states := make(map[string]llb.State, len(order))
	for _, idx := range order {
		step := &p.Steps[idx]
		def, st, err := buildStep(ctx, step, states)
		if err != nil {
			return Result{}, fmt.Errorf("building step %q: %w", step.Name, err)
		}
		result.Definitions = append(result.Definitions, def)
		result.StepNames = append(result.StepNames, step.Name)
		states[step.Name] = st
	}

	return result, nil
}

// validOrder checks that order is a complete, unique permutation of [0, n).
func validOrder(order []int, n int) bool {
	if len(order) != n {
		return false
	}
	seen := make([]bool, n)
	for _, idx := range order {
		if idx < 0 || idx >= n || seen[idx] {
			return false
		}
		seen[idx] = true
	}
	return true
}

func buildStep(ctx context.Context, step *pipeline.Step, depStates map[string]llb.State) (*llb.Definition, llb.State, error) {
	if len(step.Run) == 0 {
		return nil, llb.State{}, fmt.Errorf("step %q: %w", step.Name, pipeline.ErrMissingRun)
	}

	st := llb.Image(step.Image)

	if step.Workdir != "" {
		st = st.Dir(step.Workdir)
	}

	cmd := strings.Join(step.Run, " && ")
	runOpts := []llb.RunOption{
		llb.Args([]string{"/bin/sh", "-c", cmd}),
		llb.WithCustomName(step.Name),
	}

	for _, dep := range step.DependsOn {
		depSt, ok := depStates[dep]
		if !ok {
			return nil, llb.State{}, fmt.Errorf(
				"step %q: missing dependency state for %q", step.Name, dep,
			)
		}
		runOpts = append(runOpts, llb.AddMount(
			"/deps/"+dep,
			depSt,
			llb.Readonly,
		))
	}

	for _, m := range step.Mounts {
		runOpts = append(runOpts, llb.AddMount(
			m.Target,
			llb.Local("context"),
			llb.SourcePath(m.Source),
			llb.Readonly,
		))
	}

	for _, c := range step.Caches {
		runOpts = append(runOpts, llb.AddMount(
			c.Target,
			llb.Scratch(),
			llb.AsPersistentCacheDir(c.ID, llb.CacheMountShared),
		))
	}

	execState := st.Run(runOpts...).Root()
	def, err := execState.Marshal(ctx)
	if err != nil {
		return nil, llb.State{}, fmt.Errorf("marshaling: %w", err)
	}
	return def, execState, nil
}
