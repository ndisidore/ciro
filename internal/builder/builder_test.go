package builder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ndisidore/ciro/pkg/pipeline"
)

func TestBuild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		p         pipeline.Pipeline
		wantSteps int
		wantErr   error
	}{
		{
			name: "single step",
			p: pipeline.Pipeline{
				Name: "hello",
				Steps: []pipeline.Step{
					{
						Name:  "greet",
						Image: "alpine:latest",
						Run:   []string{"echo hello"},
					},
				},
			},
			wantSteps: 1,
		},
		{
			name: "multi step with dependency",
			p: pipeline.Pipeline{
				Name: "build",
				Steps: []pipeline.Step{
					{
						Name:  "setup",
						Image: "alpine:latest",
						Run:   []string{"echo setup"},
					},
					{
						Name:      "test",
						Image:     "golang:1.23",
						DependsOn: []string{"setup"},
						Run:       []string{"go test ./..."},
					},
				},
			},
			wantSteps: 2,
		},
		{
			name: "step with cache mount",
			p: pipeline.Pipeline{
				Name: "cached",
				Steps: []pipeline.Step{
					{
						Name:  "build",
						Image: "golang:1.23",
						Run:   []string{"go build ./..."},
						Caches: []pipeline.Cache{
							{ID: "go-build", Target: "/root/.cache/go-build"},
						},
					},
				},
			},
			wantSteps: 1,
		},
		{
			name: "step with bind mount",
			p: pipeline.Pipeline{
				Name: "mounted",
				Steps: []pipeline.Step{
					{
						Name:    "build",
						Image:   "rust:1.76",
						Run:     []string{"cargo build"},
						Workdir: "/src",
						Mounts: []pipeline.Mount{
							{Source: ".", Target: "/src"},
						},
					},
				},
			},
			wantSteps: 1,
		},
		{
			name: "multiple run commands joined",
			p: pipeline.Pipeline{
				Name: "multi-cmd",
				Steps: []pipeline.Step{
					{
						Name:  "info",
						Image: "alpine:latest",
						Run:   []string{"uname -a", "date"},
					},
				},
			},
			wantSteps: 1,
		},
		{
			name: "toposort ordering preserved",
			p: pipeline.Pipeline{
				Name: "ordered",
				Steps: []pipeline.Step{
					{
						Name:      "second",
						Image:     "alpine:latest",
						DependsOn: []string{"first"},
						Run:       []string{"echo second"},
					},
					{
						Name:  "first",
						Image: "alpine:latest",
						Run:   []string{"echo first"},
					},
				},
			},
			wantSteps: 2,
		},
		{
			name: "empty run commands",
			p: pipeline.Pipeline{
				Name: "bad",
				Steps: []pipeline.Step{
					{
						Name:  "empty",
						Image: "alpine:latest",
						Run:   nil,
					},
				},
			},
			wantErr: pipeline.ErrMissingRun,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := Build(context.Background(), tt.p)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result.Definitions, tt.wantSteps)
			assert.Len(t, result.StepNames, tt.wantSteps)
			for i, def := range result.Definitions {
				require.NotNilf(t, def, "definition[%d] is nil", i)
				assert.NotEmptyf(t, def.Def, "definition[%d] has no operations", i)
			}
		})
	}
}

func TestBuildWithPresetTopoOrder(t *testing.T) {
	t.Parallel()

	p := pipeline.Pipeline{
		Name: "cached",
		Steps: []pipeline.Step{
			{
				Name:  "a",
				Image: "alpine:latest",
				Run:   []string{"echo a"},
			},
			{
				Name:  "b",
				Image: "alpine:latest",
				Run:   []string{"echo b"},
			},
			{
				Name:  "c",
				Image: "alpine:latest",
				Run:   []string{"echo c"},
			},
		},
		TopoOrder: []int{0, 1, 2},
	}

	result, err := Build(context.Background(), p)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, result.StepNames)
	assert.Len(t, result.Definitions, 3)
}

func TestBuildTopoSortOrder(t *testing.T) {
	t.Parallel()

	p := pipeline.Pipeline{
		Name: "ordered",
		Steps: []pipeline.Step{
			{
				Name:      "c",
				Image:     "alpine:latest",
				DependsOn: []string{"b"},
				Run:       []string{"echo c"},
			},
			{
				Name:      "b",
				Image:     "alpine:latest",
				DependsOn: []string{"a"},
				Run:       []string{"echo b"},
			},
			{
				Name:  "a",
				Image: "alpine:latest",
				Run:   []string{"echo a"},
			},
		},
	}

	result, err := Build(context.Background(), p)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, result.StepNames)
}
