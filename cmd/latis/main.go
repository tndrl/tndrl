// latis is the CLI and control plane for distributed AI agents.
package main

import (
	"log"
	"os"

	"github.com/alecthomas/kong"
)

func main() {
	// First pass: parse to get --config path only
	var configPath string
	for i, arg := range os.Args[1:] {
		if arg == "-c" || arg == "--config" {
			if i+1 < len(os.Args[1:]) {
				configPath = os.Args[i+2]
			}
		} else if len(arg) > 3 && arg[:3] == "-c=" {
			configPath = arg[3:]
		} else if len(arg) > 9 && arg[:9] == "--config=" {
			configPath = arg[9:]
		}
	}

	// Load config file
	var configCLI CLI
	if err := LoadConfigFile(configPath, &configCLI); err != nil {
		log.Fatalf("failed to load config file: %v", err)
	}

	// Validate version if config was loaded
	if configPath != "" && configCLI.Version != "" {
		if err := ValidateConfigVersion(configCLI.Version); err != nil {
			log.Fatalf("config error: %v", err)
		}
	}

	// Parse CLI args into fresh struct
	var cliArgs CLI
	ctx := kong.Parse(&cliArgs,
		kong.Name("latis"),
		kong.Description("Distributed AI agent control plane"),
		kong.UsageOnError(),
	)

	// Merge config into cliArgs: config values fill in where CLI didn't set
	MergeCLIInPlace(&cliArgs, &configCLI)

	// Apply defaults for fields not set by config or CLI
	cliArgs.ApplyDefaults()

	// Resolve paths (expand ~, set defaults)
	if err := cliArgs.ResolvePaths(); err != nil {
		log.Fatalf("failed to resolve paths: %v", err)
	}

	// Run the selected command (Kong passes cliArgs to Run method)
	err := ctx.Run(&cliArgs)
	ctx.FatalIfErrorf(err)
}
