// Package main provides the CLI entry point for ciro.
package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ndisidore/ciro/pkg/parser"
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
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("fatal", slog.Any("error", err))
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
		return err
	}

	_, _ = fmt.Printf("Pipeline '%s' is valid\n", p.Name)
	_, _ = fmt.Printf("  Steps: %d\n", len(p.Steps))
	for _, s := range p.Steps {
		_, _ = fmt.Printf("    - %s (image: %s)\n", s.Name, s.Image)
	}
	return nil
}

func runAction(_ *cli.Context) error {
	_, _ = fmt.Println("run: not yet implemented")
	return nil
}
