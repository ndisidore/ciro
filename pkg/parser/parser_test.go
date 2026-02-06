package parser

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ndisidore/ciro/pkg/pipeline"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    pipeline.Pipeline
		wantErr error
	}{
		{
			name: "single step",
			input: `pipeline "hello" {
				step "greet" {
					image "alpine:latest"
					run "echo hello"
				}
			}`,
			want: pipeline.Pipeline{
				Name: "hello",
				Steps: []pipeline.Step{
					{
						Name:  "greet",
						Image: "alpine:latest",
						Run:   []string{"echo hello"},
					},
				},
			},
		},
		{
			name: "multi step with dependencies",
			input: `pipeline "build" {
				step "setup" {
					image "alpine:latest"
					run "echo setup"
				}
				step "test" {
					image "golang:1.23"
					depends-on "setup"
					run "go test ./..."
				}
			}`,
			want: pipeline.Pipeline{
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
		},
		{
			name: "step with mount and cache",
			input: `pipeline "full" {
				step "build" {
					image "rust:1.76"
					mount "." "/src"
					cache "cargo" "/root/.cargo"
					workdir "/src"
					run "cargo build"
				}
			}`,
			want: pipeline.Pipeline{
				Name: "full",
				Steps: []pipeline.Step{
					{
						Name:    "build",
						Image:   "rust:1.76",
						Workdir: "/src",
						Run:     []string{"cargo build"},
						Mounts:  []pipeline.Mount{{Source: ".", Target: "/src"}},
						Caches:  []pipeline.Cache{{ID: "cargo", Target: "/root/.cargo"}},
					},
				},
			},
		},
		{
			name: "multiple run commands",
			input: `pipeline "multi" {
				step "info" {
					image "alpine:latest"
					run "uname -a"
					run "date"
				}
			}`,
			want: pipeline.Pipeline{
				Name: "multi",
				Steps: []pipeline.Step{
					{
						Name:  "info",
						Image: "alpine:latest",
						Run:   []string{"uname -a", "date"},
					},
				},
			},
		},
		{
			name: "missing pipeline node",
			input: `something "foo" {
				image "alpine:latest"
			}`,
			wantErr: ErrNoPipeline,
		},
		{
			name: "missing image",
			input: `pipeline "bad" {
				step "noimg" {
					run "echo oops"
				}
			}`,
			wantErr: pipeline.ErrMissingImage,
		},
		{
			name: "duplicate step names",
			input: `pipeline "dup" {
				step "a" {
					image "alpine:latest"
					run "echo 1"
				}
				step "a" {
					image "alpine:latest"
					run "echo 2"
				}
			}`,
			wantErr: pipeline.ErrDuplicateStep,
		},
		{
			name: "unknown dependency",
			input: `pipeline "bad" {
				step "a" {
					image "alpine:latest"
					depends-on "nonexistent"
					run "echo 1"
				}
			}`,
			wantErr: pipeline.ErrUnknownDep,
		},
		{
			name: "pipeline without name",
			input: `pipeline {
				step "a" {
					image "alpine:latest"
					run "echo hi"
				}
			}`,
			wantErr: ErrMissingName,
		},
		{
			name: "empty pipeline",
			input: `pipeline "empty" {
			}`,
			wantErr: pipeline.ErrEmptyPipeline,
		},
		{
			name: "unknown step child node",
			input: `pipeline "bad" {
				step "a" {
					image "alpine:latest"
					run "echo hi"
					foobar "wat"
				}
			}`,
			wantErr: ErrUnknownNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseString(tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error wrapping %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error wrapping %v, got: %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
