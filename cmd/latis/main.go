// latis is the CLI and control plane for distributed AI agents.
package main

import (
	"log"
	"os"

	"github.com/alecthomas/kong"
)

func main() {
	cli := CLI{}

	// First pass: parse to get --config path
	parser, err := kong.New(&cli,
		kong.Name("latis"),
		kong.Description("Distributed AI agent control plane"),
	)
	if err != nil {
		log.Fatalf("failed to create parser: %v", err)
	}

	// First pass ignores errors (we just need the config path)
	_, _ = parser.Parse(os.Args[1:])

	// Load config file (if provided)
	if err := LoadConfigFile(cli.Config, &cli); err != nil {
		log.Fatalf("failed to load config file: %v", err)
	}

	// Validate version if config was loaded
	if cli.Config != "" && cli.Version != "" {
		if err := ValidateConfigVersion(cli.Version); err != nil {
			log.Fatalf("config error: %v", err)
		}
	}

	// Second pass: CLI/env override file values, run subcommand
	ctx := kong.Parse(&cli,
		kong.Name("latis"),
		kong.Description("Distributed AI agent control plane"),
		kong.UsageOnError(),
	)

	// Resolve paths (expand ~, set defaults)
	if err := cli.ResolvePaths(); err != nil {
		log.Fatalf("failed to resolve paths: %v", err)
	}

	// Run the selected command
	err = ctx.Run(&cli)
	ctx.FatalIfErrorf(err)
}
