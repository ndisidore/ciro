package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSolver implements the Solver interface for testing.
type fakeSolver struct {
	solveFn func(ctx context.Context, def *llb.Definition, opt client.SolveOpt, ch chan *client.SolveStatus) (*client.SolveResponse, error)
}

func (f *fakeSolver) Solve(ctx context.Context, def *llb.Definition, opt client.SolveOpt, ch chan *client.SolveStatus) (*client.SolveResponse, error) {
	return f.solveFn(ctx, def, opt, ch)
}

// fakeDisplay implements progress.Display for testing.
type fakeDisplay struct {
	runFn func(ctx context.Context, name string, ch <-chan *client.SolveStatus) error
}

func (f *fakeDisplay) Run(ctx context.Context, name string, ch <-chan *client.SolveStatus) error {
	if f.runFn != nil {
		return f.runFn(ctx, name, ch)
	}
	for s := range ch {
		_ = s
	}
	return nil
}

func TestRun(t *testing.T) {
	t.Parallel()

	// Pre-marshal a minimal definition to reuse across test cases.
	def, err := llb.Scratch().Marshal(context.Background())
	require.NoError(t, err)

	tests := []struct {
		name         string
		input        RunInput
		wantErr      string
		wantSentinel error
	}{
		{
			name: "single step solves successfully",
			input: RunInput{
				Solver: &fakeSolver{solveFn: func(_ context.Context, _ *llb.Definition, _ client.SolveOpt, ch chan *client.SolveStatus) (*client.SolveResponse, error) {
					close(ch)
					return &client.SolveResponse{}, nil
				}},
				Steps:   []Step{{Name: "build", Definition: def}},
				Display: &fakeDisplay{},
			},
		},
		{
			name: "multi-step sequential execution",
			input: RunInput{
				Solver: &fakeSolver{solveFn: func(_ context.Context, _ *llb.Definition, _ client.SolveOpt, ch chan *client.SolveStatus) (*client.SolveResponse, error) {
					close(ch)
					return &client.SolveResponse{}, nil
				}},
				Steps: []Step{
					{Name: "first", Definition: def},
					{Name: "second", Definition: def},
					{Name: "third", Definition: def},
				},
				Display: &fakeDisplay{},
			},
		},
		{
			name: "solve error wraps with step name",
			input: RunInput{
				Solver: &fakeSolver{solveFn: func(_ context.Context, _ *llb.Definition, _ client.SolveOpt, ch chan *client.SolveStatus) (*client.SolveResponse, error) {
					close(ch)
					return nil, errors.New("connection refused")
				}},
				Steps:   []Step{{Name: "deploy", Definition: def}},
				Display: &fakeDisplay{},
			},
			wantErr: `step "deploy"`,
		},
		{
			name: "display error propagates",
			input: RunInput{
				Solver: &fakeSolver{solveFn: func(_ context.Context, _ *llb.Definition, _ client.SolveOpt, ch chan *client.SolveStatus) (*client.SolveResponse, error) {
					close(ch)
					return &client.SolveResponse{}, nil
				}},
				Steps: []Step{{Name: "render", Definition: def}},
				Display: &fakeDisplay{runFn: func(_ context.Context, _ string, ch <-chan *client.SolveStatus) error {
					for s := range ch {
						_ = s
					}
					return errors.New("terminal error")
				}},
			},
			wantErr: "displaying progress",
		},
		{
			name: "empty steps is a no-op",
			input: RunInput{
				Solver:  &fakeSolver{},
				Steps:   []Step{},
				Display: &fakeDisplay{},
			},
		},
		{
			name: "nil solver returns error",
			input: RunInput{
				Solver:  nil,
				Steps:   []Step{{Name: "x", Definition: def}},
				Display: &fakeDisplay{},
			},
			wantSentinel: ErrNilSolver,
		},
		{
			name: "nil display returns error",
			input: RunInput{
				Solver:  &fakeSolver{},
				Steps:   []Step{{Name: "x", Definition: def}},
				Display: nil,
			},
			wantSentinel: ErrNilDisplay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := Run(context.Background(), tt.input)
			if tt.wantSentinel != nil {
				require.ErrorIs(t, err, tt.wantSentinel)
				return
			}
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRun_contextCancellation(t *testing.T) {
	t.Parallel()

	def, err := llb.Scratch().Marshal(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err = Run(ctx, RunInput{
		Solver: &fakeSolver{solveFn: func(ctx context.Context, _ *llb.Definition, _ client.SolveOpt, ch chan *client.SolveStatus) (*client.SolveResponse, error) {
			close(ch)
			return nil, ctx.Err()
		}},
		Steps:   []Step{{Name: "cancelled", Definition: def}},
		Display: &fakeDisplay{},
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestRun_stepOrder(t *testing.T) {
	t.Parallel()

	def, err := llb.Scratch().Marshal(context.Background())
	require.NoError(t, err)

	var order []string
	solver := &fakeSolver{solveFn: func(_ context.Context, _ *llb.Definition, _ client.SolveOpt, ch chan *client.SolveStatus) (*client.SolveResponse, error) {
		close(ch)
		return &client.SolveResponse{}, nil
	}}
	display := &fakeDisplay{runFn: func(_ context.Context, name string, ch <-chan *client.SolveStatus) error {
		order = append(order, name)
		for s := range ch {
			_ = s
		}
		return nil
	}}

	err = Run(context.Background(), RunInput{
		Solver: solver,
		Steps: []Step{
			{Name: "a", Definition: def},
			{Name: "b", Definition: def},
			{Name: "c", Definition: def},
		},
		Display: display,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, order)
}

func TestRun_nilDefinition(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), RunInput{
		Solver:  &fakeSolver{},
		Steps:   []Step{{Name: "bad", Definition: nil}},
		Display: &fakeDisplay{},
	})
	require.ErrorIs(t, err, ErrNilDefinition)
}

func TestRun_solverWritesStatus(t *testing.T) {
	t.Parallel()

	def, err := llb.Scratch().Marshal(context.Background())
	require.NoError(t, err)

	var received int
	solver := &fakeSolver{solveFn: func(_ context.Context, _ *llb.Definition, _ client.SolveOpt, ch chan *client.SolveStatus) (*client.SolveResponse, error) {
		ch <- &client.SolveStatus{}
		ch <- &client.SolveStatus{}
		close(ch)
		return &client.SolveResponse{}, nil
	}}
	display := &fakeDisplay{runFn: func(_ context.Context, _ string, ch <-chan *client.SolveStatus) error {
		for s := range ch {
			_ = s
			received++
		}
		return nil
	}}

	err = Run(context.Background(), RunInput{
		Solver:  solver,
		Steps:   []Step{{Name: "status-test", Definition: def}},
		Display: display,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, received)
}
