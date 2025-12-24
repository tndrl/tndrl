// latis is the CLI and control plane for distributed AI agents.
package main

import (
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
)

func main() {
	// First pass: parse to get --config path and --log-level
	var configPath, logLevel string
	var verbose bool
	for i, arg := range os.Args[1:] {
		if arg == "-c" || arg == "--config" {
			if i+1 < len(os.Args[1:]) {
				configPath = os.Args[i+2]
			}
		} else if len(arg) > 3 && arg[:3] == "-c=" {
			configPath = arg[3:]
		} else if len(arg) > 9 && arg[:9] == "--config=" {
			configPath = arg[9:]
		} else if len(arg) > 12 && arg[:12] == "--log-level=" {
			logLevel = arg[12:]
		} else if arg == "-v" || arg == "--verbose" {
			verbose = true
		}
	}

	// Setup logger early (before any logging happens)
	if verbose {
		logLevel = "debug"
	} else if logLevel == "" {
		logLevel = "info"
	}
	setupLogger(logLevel)

	// Load config file
	var configCLI CLI
	if err := LoadConfigFile(configPath, &configCLI); err != nil {
		slog.Error("failed to load config file", "err", err)
		os.Exit(1)
	}

	// Validate version if config was loaded
	if configPath != "" && configCLI.Version != "" {
		if err := ValidateConfigVersion(configCLI.Version); err != nil {
			slog.Error("config error", "err", err)
			os.Exit(1)
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

	// Re-setup logger with final config (in case it came from config file)
	level := cliArgs.LogLevel
	if cliArgs.Verbose {
		level = "debug"
	}
	setupLogger(level)

	// Resolve paths (expand ~, set defaults)
	if err := cliArgs.ResolvePaths(); err != nil {
		slog.Error("failed to resolve paths", "err", err)
		os.Exit(1)
	}

	// Run the selected command (Kong passes cliArgs to Run method)
	err := ctx.Run(&cliArgs)
	ctx.FatalIfErrorf(err)
}
