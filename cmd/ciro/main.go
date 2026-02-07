// Package main provides the CLI entry point for ciro.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ndisidore/ciro/internal/builder"
	"github.com/ndisidore/ciro/pkg/parser"
	"github.com/ndisidore/ciro/pkg/pipeline"
)

func main() {
	app := &cli.App{
		Name:  "ciro",
		Usage: "a container-native CI/CD pipeline runner",
		Commands: []*cli.Command{
			{
				Name:      "validate",
				Usage:     "validate a KDL pipeline file",
				ArgsUsage: "<file>",
				Action:    validateAction,
			},
			{
				Name:      "run",
				Usage:     "run a KDL pipeline against BuildKit",
				ArgsUsage: "<file>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "addr",
						Usage: "BuildKit daemon address",
						Value: "tcp://127.0.0.1:1234",
					},
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "generate LLB without executing",
					},
				},
				Action: runAction,
			},
		},
		ExitErrHandler: func(_ *cli.Context, err error) {
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		},
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

func validateAction(c *cli.Context) error {
	path := c.Args().First()
	if path == "" {
		return errors.New("usage: ciro validate <file>")
	}

	p, err := parser.ParseFile(path)
	if err != nil {
		return fmt.Errorf("validating %s: %w", path, err)
	}

	printPipelineSummary(p.Name, p.Steps)
	return nil
}

func runAction(c *cli.Context) error {
	path := c.Args().First()
	if path == "" {
		return errors.New("usage: ciro run <file>")
	}

	p, err := parser.ParseFile(path)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	if !c.Bool("dry-run") {
		return errors.New("live execution not yet implemented (use --dry-run)")
	}

	result, err := builder.Build(c.Context, p)
	if err != nil {
		return fmt.Errorf("building %s: %w", path, err)
	}

	_, _ = fmt.Printf("Pipeline '%s' validated\n", p.Name)
	_, _ = fmt.Printf("  Steps: %d\n", len(result.Definitions))
	for i, name := range result.StepNames {
		ops := 0
		if i < len(result.Definitions) && result.Definitions[i] != nil {
			ops = len(result.Definitions[i].Def)
		}
		_, _ = fmt.Printf("    - %s (%d LLB ops)\n", name, ops)
	}
	return nil
}

func printPipelineSummary(name string, steps []pipeline.Step) {
	_, _ = fmt.Printf("Pipeline '%s' is valid\n", name)
	_, _ = fmt.Printf("  Steps: %d\n", len(steps))
	for _, s := range steps {
		_, _ = fmt.Printf("    - %s (image: %s)\n", s.Name, s.Image)
	}
}
