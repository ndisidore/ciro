// Package parser converts KDL documents into pipeline definitions.
package parser

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	kdl "github.com/sblinch/kdl-go"
	"github.com/sblinch/kdl-go/document"

	"github.com/ndisidore/ciro/pkg/pipeline"
)

// Sentinel errors for parse failures.
var (
	ErrNoPipeline   = errors.New("no pipeline node found")
	ErrMissingName  = errors.New("pipeline node missing name argument")
	ErrUnknownNode  = errors.New("unknown node type")
	ErrMissingField = errors.New("missing required field")
)

// ParseFile reads and parses a KDL pipeline file at the given path.
func ParseFile(path string) (pipeline.Pipeline, error) {
	f, err := os.Open(path)
	if err != nil {
		return pipeline.Pipeline{}, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	return Parse(f, path)
}

// Parse parses KDL content from the reader into a Pipeline.
func Parse(r io.Reader, filename string) (pipeline.Pipeline, error) {
	doc, err := kdl.Parse(r)
	if err != nil {
		return pipeline.Pipeline{}, fmt.Errorf("parsing %s: %w", filename, err)
	}

	for _, node := range doc.Nodes {
		if node.Name.ValueString() == "pipeline" {
			return parsePipeline(node, filename)
		}
	}

	return pipeline.Pipeline{}, fmt.Errorf("%s: %w", filename, ErrNoPipeline)
}

// ParseString parses KDL content from a string into a Pipeline.
func ParseString(content string) (pipeline.Pipeline, error) {
	return Parse(strings.NewReader(content), "<string>")
}

func parsePipeline(node *document.Node, filename string) (pipeline.Pipeline, error) {
	name, err := stringArg(node, 0)
	if err != nil {
		return pipeline.Pipeline{}, fmt.Errorf(
			"%s: pipeline name: %w", filename, ErrMissingName,
		)
	}

	p := pipeline.Pipeline{
		Name:  name,
		Steps: make([]pipeline.Step, 0, len(node.Children)),
	}

	for _, child := range node.Children {
		if childName := child.Name.ValueString(); childName != "step" {
			return pipeline.Pipeline{}, fmt.Errorf(
				"%s: %w: %q (expected step)", filename, ErrUnknownNode, childName,
			)
		}

		step, err := parseStep(child, filename)
		if err != nil {
			return pipeline.Pipeline{}, err
		}
		p.Steps = append(p.Steps, step)
	}

	if err := p.Validate(); err != nil {
		return pipeline.Pipeline{}, fmt.Errorf("%s: %w", filename, err)
	}

	return p, nil
}

func parseStep(node *document.Node, filename string) (pipeline.Step, error) {
	name, err := stringArg(node, 0)
	if err != nil {
		return pipeline.Step{}, fmt.Errorf(
			"%s: step missing name: %w", filename, ErrMissingField,
		)
	}

	s := pipeline.Step{Name: name}
	for _, child := range node.Children {
		if err := applyStepField(&s, child, filename); err != nil {
			return pipeline.Step{}, err
		}
	}
	return s, nil
}

func applyStepField(s *pipeline.Step, node *document.Node, filename string) error {
	childName := node.Name.ValueString()

	// Single-arg fields share the same extraction pattern.
	switch childName {
	case "image", "run", "workdir", "depends-on":
		v, err := requireStringArg(node, filename, childName)
		if err != nil {
			return err
		}
		setSingleArg(s, childName, v)
		return nil
	case "mount":
		m, err := stringArgs2(node, filename, "mount")
		if err != nil {
			return err
		}
		s.Mounts = append(s.Mounts, pipeline.Mount{Source: m[0], Target: m[1]})
		return nil
	case "cache":
		c, err := stringArgs2(node, filename, "cache")
		if err != nil {
			return err
		}
		s.Caches = append(s.Caches, pipeline.Cache{ID: c[0], Target: c[1]})
		return nil
	default:
		return fmt.Errorf(
			"%s: step %q: %w: %q", filename, s.Name, ErrUnknownNode, childName,
		)
	}
}

func setSingleArg(s *pipeline.Step, field, value string) {
	switch field {
	case "image":
		s.Image = value
	case "run":
		s.Run = append(s.Run, value)
	case "workdir":
		s.Workdir = value
	case "depends-on":
		s.DependsOn = append(s.DependsOn, value)
	default:
		// Unreachable: only called from applyStepField with known fields.
	}
}

// stringArgs2 extracts exactly two string arguments from a node.
func stringArgs2(node *document.Node, filename, field string) ([2]string, error) {
	first, err := stringArg(node, 0)
	if err != nil {
		return [2]string{}, fmt.Errorf(
			"%s: %s requires two arguments: %w", filename, field, ErrMissingField,
		)
	}
	second, err := stringArg(node, 1)
	if err != nil {
		return [2]string{}, fmt.Errorf(
			"%s: %s requires two arguments: %w", filename, field, ErrMissingField,
		)
	}
	return [2]string{first, second}, nil
}

// requireStringArg extracts the first string argument, wrapping errors with context.
func requireStringArg(node *document.Node, filename, field string) (string, error) {
	v, err := stringArg(node, 0)
	if err != nil {
		return "", fmt.Errorf(
			"%s: %q requires a string value: %w", filename, field, ErrMissingField,
		)
	}
	return v, nil
}

// stringArg returns the string value at the given argument index, or an error.
func stringArg(node *document.Node, idx int) (string, error) {
	if idx >= len(node.Arguments) {
		return "", fmt.Errorf("argument %d: %w", idx, ErrMissingField)
	}
	v, ok := node.Arguments[idx].ResolvedValue().(string)
	if !ok {
		return "", fmt.Errorf("argument %d: not a string: %w", idx, ErrMissingField)
	}
	return v, nil
}
