package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectImages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    Pipeline
		want []string
	}{
		{
			name: "unique images",
			p: Pipeline{
				Steps: []Step{
					{Image: "alpine:latest"},
					{Image: "golang:1.23"},
					{Image: "rust:1.76"},
				},
			},
			want: []string{"alpine:latest", "golang:1.23", "rust:1.76"},
		},
		{
			name: "deduplicates",
			p: Pipeline{
				Steps: []Step{
					{Image: "alpine:latest"},
					{Image: "alpine:latest"},
					{Image: "golang:1.23"},
				},
			},
			want: []string{"alpine:latest", "golang:1.23"},
		},
		{
			name: "empty pipeline",
			p:    Pipeline{},
			want: []string{},
		},
		{
			name: "single step",
			p: Pipeline{
				Steps: []Step{
					{Image: "ubuntu:22.04"},
				},
			},
			want: []string{"ubuntu:22.04"},
		},
		{
			name: "skips empty image refs",
			p: Pipeline{
				Steps: []Step{
					{Image: "alpine:latest"},
					{Image: ""},
					{Image: "golang:1.23"},
				},
			},
			want: []string{"alpine:latest", "golang:1.23"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CollectImages(tt.p)
			assert.Equal(t, tt.want, got)
		})
	}
}
