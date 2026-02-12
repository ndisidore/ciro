package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterSteps(t *testing.T) {
	t.Parallel()

	// Helper to build a minimal step with optional deps and exports.
	step := func(name string, deps []string, exports []Export) Step {
		return Step{
			Name:      name,
			Image:     "alpine",
			Run:       []string{"echo " + name},
			DependsOn: deps,
			Exports:   exports,
		}
	}

	// Linear chain: lint -> build -> test -> deploy
	linearChain := []Step{
		step("lint", nil, nil),
		step("build", []string{"lint"}, nil),
		step("test", []string{"build"}, nil),
		step("deploy", []string{"test"}, nil),
	}

	// Diamond DAG:
	//   lint
	//  /    \
	// build  check
	//  \    /
	//  deploy
	diamondDAG := []Step{
		step("lint", nil, nil),
		step("build", []string{"lint"}, nil),
		step("check", []string{"lint"}, nil),
		step("deploy", []string{"build", "check"}, nil),
	}

	// Branching DAG with exports:
	//   a -> b -> c
	//   d -> e
	branchingDAG := []Step{
		step("a", nil, []Export{{Path: "/out/a", Local: "./a"}}),
		step("b", []string{"a"}, []Export{{Path: "/out/b", Local: "./b"}}),
		step("c", []string{"b"}, []Export{{Path: "/out/c", Local: "./c"}}),
		step("d", nil, []Export{{Path: "/out/d", Local: "./d"}}),
		step("e", []string{"d"}, []Export{{Path: "/out/e", Local: "./e"}}),
	}

	tests := []struct {
		name    string
		steps   []Step
		opts    FilterOpts
		want    []Step
		wantErr error
	}{
		{
			name:  "no flags identity",
			steps: linearChain,
			opts:  FilterOpts{},
			want:  linearChain,
		},
		{
			name:  "start-at first step is identity",
			steps: linearChain,
			opts:  FilterOpts{StartAt: "lint"},
			want:  linearChain,
		},
		{
			name:  "start-at mid step linear",
			steps: linearChain,
			opts:  FilterOpts{StartAt: "build"},
			want: []Step{
				step("lint", nil, nil),
				step("build", []string{"lint"}, nil),
				step("test", []string{"build"}, nil),
				step("deploy", []string{"test"}, nil),
			},
		},
		{
			name:  "start-at last step linear",
			steps: linearChain,
			opts:  FilterOpts{StartAt: "deploy"},
			want: []Step{
				step("lint", nil, nil),
				step("build", []string{"lint"}, nil),
				step("test", []string{"build"}, nil),
				step("deploy", []string{"test"}, nil),
			},
		},
		{
			name:  "stop-after first step linear",
			steps: linearChain,
			opts:  FilterOpts{StopAfter: "lint"},
			want: []Step{
				step("lint", nil, nil),
			},
		},
		{
			name:  "stop-after mid step linear",
			steps: linearChain,
			opts:  FilterOpts{StopAfter: "test"},
			want: []Step{
				step("lint", nil, nil),
				step("build", []string{"lint"}, nil),
				step("test", []string{"build"}, nil),
			},
		},
		{
			name:  "stop-after last step is identity",
			steps: linearChain,
			opts:  FilterOpts{StopAfter: "deploy"},
			want:  linearChain,
		},
		{
			name:  "both flags same step",
			steps: linearChain,
			opts:  FilterOpts{StartAt: "build", StopAfter: "build"},
			want: []Step{
				step("lint", nil, nil),
				step("build", []string{"lint"}, nil),
			},
		},
		{
			name:  "both flags window",
			steps: linearChain,
			opts:  FilterOpts{StartAt: "build", StopAfter: "test"},
			want: []Step{
				step("lint", nil, nil),
				step("build", []string{"lint"}, nil),
				step("test", []string{"build"}, nil),
			},
		},
		{
			name:  "diamond start-at one branch includes sibling deps",
			steps: diamondDAG,
			opts:  FilterOpts{StartAt: "build"},
			want: []Step{
				step("lint", nil, nil),
				step("build", []string{"lint"}, nil),
				step("check", []string{"lint"}, nil),
				step("deploy", []string{"build", "check"}, nil),
			},
		},
		{
			name:  "diamond stop-after mid",
			steps: diamondDAG,
			opts:  FilterOpts{StopAfter: "build"},
			want: []Step{
				step("lint", nil, nil),
				step("build", []string{"lint"}, nil),
			},
		},
		{
			name:  "diamond start-at deploy includes all deps",
			steps: diamondDAG,
			opts:  FilterOpts{StartAt: "deploy"},
			want:  diamondDAG,
		},
		{
			name:  "branching start-at only one branch",
			steps: branchingDAG,
			opts:  FilterOpts{StartAt: "d"},
			want: []Step{
				step("d", nil, []Export{{Path: "/out/d", Local: "./d"}}),
				step("e", []string{"d"}, []Export{{Path: "/out/e", Local: "./e"}}),
			},
		},
		{
			name:  "branching stop-after only one branch",
			steps: branchingDAG,
			opts:  FilterOpts{StopAfter: "b"},
			want: []Step{
				step("a", nil, []Export{{Path: "/out/a", Local: "./a"}}),
				step("b", []string{"a"}, []Export{{Path: "/out/b", Local: "./b"}}),
			},
		},
		{
			name:    "disjoint branches empty result",
			steps:   branchingDAG,
			opts:    FilterOpts{StartAt: "d", StopAfter: "b"},
			wantErr: ErrEmptyFilterResult,
		},
		{
			name: "single step pipeline",
			steps: []Step{
				step("only", nil, nil),
			},
			opts: FilterOpts{StartAt: "only"},
			want: []Step{
				step("only", nil, nil),
			},
		},
		{
			name: "single step stop-after",
			steps: []Step{
				step("only", nil, nil),
			},
			opts: FilterOpts{StopAfter: "only"},
			want: []Step{
				step("only", nil, nil),
			},
		},
		{
			name: "independent steps start-at",
			steps: []Step{
				step("a", nil, nil),
				step("b", nil, nil),
				step("c", nil, nil),
			},
			opts: FilterOpts{StartAt: "b"},
			want: []Step{
				step("b", nil, nil),
			},
		},
		{
			name: "independent steps stop-after",
			steps: []Step{
				step("a", nil, nil),
				step("b", nil, nil),
				step("c", nil, nil),
			},
			opts: FilterOpts{StopAfter: "b"},
			want: []Step{
				step("b", nil, nil),
			},
		},
		{
			name:    "unknown start-at",
			steps:   linearChain,
			opts:    FilterOpts{StartAt: "nonexistent"},
			wantErr: ErrUnknownStartAt,
		},
		{
			name:    "unknown stop-after",
			steps:   linearChain,
			opts:    FilterOpts{StopAfter: "nonexistent"},
			wantErr: ErrUnknownStopAfter,
		},
		{
			name: "matrix-expanded names with brackets",
			steps: []Step{
				step("lint", nil, nil),
				step("build[os=linux]", []string{"lint"}, nil),
				step("build[os=darwin]", []string{"lint"}, nil),
				step("test[os=linux]", []string{"build[os=linux]"}, nil),
				step("test[os=darwin]", []string{"build[os=darwin]"}, nil),
			},
			opts: FilterOpts{StartAt: "build[os=linux]"},
			want: []Step{
				step("lint", nil, nil),
				step("build[os=linux]", []string{"lint"}, nil),
				step("test[os=linux]", []string{"build[os=linux]"}, nil),
			},
		},
		{
			name: "export stripping on dep-only steps",
			steps: []Step{
				step("a", nil, []Export{{Path: "/out/a", Local: "./a"}}),
				step("b", []string{"a"}, []Export{{Path: "/out/b", Local: "./b"}}),
				step("c", []string{"b"}, []Export{{Path: "/out/c", Local: "./c"}}),
			},
			opts: FilterOpts{StartAt: "b"},
			want: []Step{
				{Name: "a", Image: "alpine", Run: []string{"echo a"}, DependsOn: nil, Exports: nil},
				step("b", []string{"a"}, []Export{{Path: "/out/b", Local: "./b"}}),
				step("c", []string{"b"}, []Export{{Path: "/out/c", Local: "./c"}}),
			},
		},
		{
			name: "export stripping with stop-after",
			steps: []Step{
				step("a", nil, []Export{{Path: "/out/a", Local: "./a"}}),
				step("b", []string{"a"}, []Export{{Path: "/out/b", Local: "./b"}}),
				step("c", []string{"b"}, []Export{{Path: "/out/c", Local: "./c"}}),
			},
			opts: FilterOpts{StartAt: "c", StopAfter: "c"},
			want: []Step{
				{Name: "a", Image: "alpine", Run: []string{"echo a"}, DependsOn: nil, Exports: nil},
				{Name: "b", Image: "alpine", Run: []string{"echo b"}, DependsOn: []string{"a"}, Exports: nil},
				step("c", []string{"b"}, []Export{{Path: "/out/c", Local: "./c"}}),
			},
		},
		{
			name: "preserves original slice order",
			steps: []Step{
				step("z", nil, nil),
				step("m", []string{"z"}, nil),
				step("a", []string{"m"}, nil),
			},
			opts: FilterOpts{StopAfter: "m"},
			want: []Step{
				step("z", nil, nil),
				step("m", []string{"z"}, nil),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := FilterSteps(tt.steps, tt.opts)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
