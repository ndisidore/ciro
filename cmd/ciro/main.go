// Package main provides the CLI entry point for ciro.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v2"
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
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func validateAction(_ *cli.Context) error {
	fmt.Println("validate: not yet implemented")
	return nil
}

func runAction(_ *cli.Context) error {
	fmt.Println("run: not yet implemented")
	return nil
}
