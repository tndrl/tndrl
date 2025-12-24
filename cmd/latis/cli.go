package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/shanemcd/latis/pkg/llm"
	"github.com/shanemcd/latis/pkg/pki"
)

// ConfigVersion is the current config file version.
const ConfigVersion = "v1"

// CLI is the root command structure for latis.
// It serves as the single source of truth for CLI flags, env vars, and config files.
type CLI struct {
	// Global flags (shared across all subcommands)
	Config   string `short:"c" help:"Path to config file" type:"path" yaml:"-"`
	LogLevel string `help:"Log level (debug, info, warn, error)" default:"info" env:"LATIS_LOG_LEVEL" yaml:"logLevel"`
	Verbose  bool   `short:"v" help:"Verbose output (same as --log-level=debug)" yaml:"-"`

	// Embedded config (populated from file + CLI + env)
	Version string       `yaml:"version" kong:"-"`
	Server  ServerConfig `embed:"" prefix:"server-" yaml:"server"`
	Agent   AgentConfig  `embed:"" prefix:"agent-" yaml:"agent"`
	LLM     LLMConfig    `embed:"" prefix:"llm-" yaml:"llm"`
	PKI     PKIConfig    `embed:"" prefix:"pki-" yaml:"pki"`
	Peers   []PeerConfig `yaml:"peers" kong:"-"`

	// Subcommands
	Serve    ServeCmd    `cmd:"" help:"Run as daemon (listen for connections)"`
	Ping     PingCmd     `cmd:"" help:"Ping a peer"`
	Status   StatusCmd   `cmd:"" help:"Get peer status"`
	Prompt   PromptCmd   `cmd:"" help:"Send prompt to peer"`
	Discover DiscoverCmd `cmd:"" help:"Discover peer capabilities (AgentCard)"`
	Shutdown ShutdownCmd `cmd:"" help:"Request peer shutdown"`
}

// ServerConfig holds server-mode configuration.
type ServerConfig struct {
	Addr string `help:"Address to listen on" env:"LATIS_ADDR" yaml:"addr"`
}

// AgentConfig holds A2A agent card configuration.
type AgentConfig struct {
	Name        string   `help:"Agent name" env:"LATIS_AGENT_NAME" yaml:"name"`
	Description string   `help:"Agent description" env:"LATIS_AGENT_DESCRIPTION" yaml:"description"`
	InputModes  []string `help:"Supported input modes" env:"LATIS_AGENT_INPUT_MODES" yaml:"inputModes"`
	OutputModes []string `help:"Supported output modes" env:"LATIS_AGENT_OUTPUT_MODES" yaml:"outputModes"`
	Streaming   *bool    `help:"Enable streaming" env:"LATIS_AGENT_STREAMING" yaml:"streaming"`
	Skills      []Skill  `yaml:"skills" kong:"-"`
}

// Skill represents an A2A agent skill.
type Skill struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Examples    []string `yaml:"examples"`
}

// LLMConfig holds LLM provider configuration.
type LLMConfig struct {
	Provider string `help:"LLM provider (echo, ollama)" env:"LATIS_LLM_PROVIDER" yaml:"provider"`
	Model    string `help:"LLM model name" env:"LATIS_LLM_MODEL" yaml:"model"`
	URL      string `help:"LLM API URL" env:"LATIS_LLM_URL" yaml:"url"`
}

// PKIConfig holds PKI-related configuration.
type PKIConfig struct {
	Dir    string `help:"PKI directory" env:"LATIS_PKI_DIR" yaml:"dir"`
	CACert string `help:"CA certificate path (overrides pki-dir)" env:"LATIS_CA_CERT" yaml:"caCert"`
	CAKey  string `help:"CA private key path" env:"LATIS_CA_KEY" yaml:"caKey"`
	Cert   string `help:"Certificate path" env:"LATIS_CERT" yaml:"cert"`
	Key    string `help:"Private key path" env:"LATIS_KEY" yaml:"key"`
	Init   bool   `help:"Initialize PKI if missing" env:"LATIS_INIT_PKI" yaml:"init"`
}

// PeerConfig holds configuration for a known peer.
type PeerConfig struct {
	Name string `yaml:"name"`
	Addr string `yaml:"addr"`
}

// LoadConfigFile loads configuration from a YAML file into the CLI struct.
// If the path is empty, this is a no-op.
func LoadConfigFile(path string, cli *CLI) error {
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cli); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	return nil
}

// ApplyDefaults sets default values for fields that weren't set by config or CLI.
func (cli *CLI) ApplyDefaults() {
	if cli.Server.Addr == "" {
		cli.Server.Addr = "[::]:4433"
	}
	if cli.PKI.Dir == "" {
		cli.PKI.Dir = "~/.latis/pki"
	}
	if len(cli.Agent.InputModes) == 0 {
		cli.Agent.InputModes = []string{"text"}
	}
	if len(cli.Agent.OutputModes) == 0 {
		cli.Agent.OutputModes = []string{"text"}
	}
	if cli.Agent.Streaming == nil {
		t := true
		cli.Agent.Streaming = &t
	}
}

// IsStreaming returns whether streaming is enabled (defaults to true).
func (cli *CLI) IsStreaming() bool {
	if cli.Agent.Streaming == nil {
		return true
	}
	return *cli.Agent.Streaming
}

// MergeCLIInPlace fills in dst with values from config where dst has zero values.
// This implements: CLI > config (CLI values are preserved, config fills gaps).
func MergeCLIInPlace(dst, config *CLI) {
	mergeStructsInPlace(reflect.ValueOf(dst).Elem(), reflect.ValueOf(config).Elem())
}

// mergeStructsInPlace recursively fills dst with config values where dst is zero.
func mergeStructsInPlace(dst, config reflect.Value) {
	for i := 0; i < dst.NumField(); i++ {
		dstField := dst.Field(i)
		configField := config.Field(i)

		if !dstField.CanSet() {
			continue
		}

		switch dstField.Kind() {
		case reflect.Struct:
			mergeStructsInPlace(dstField, configField)
		case reflect.Ptr:
			if dstField.IsNil() && !configField.IsNil() {
				dstField.Set(configField)
			}
		case reflect.Slice:
			if dstField.Len() == 0 && configField.Len() > 0 {
				dstField.Set(configField)
			}
		case reflect.String:
			if dstField.String() == "" && configField.String() != "" {
				dstField.Set(configField)
			}
		case reflect.Bool:
			// For bools, config true fills in if dst is false
			if !dstField.Bool() && configField.Bool() {
				dstField.Set(configField)
			}
		default:
			if dstField.IsZero() && !configField.IsZero() {
				dstField.Set(configField)
			}
		}
	}
}

// ValidateConfigVersion checks that the config file version is supported.
func ValidateConfigVersion(version string) error {
	if version == "" {
		return fmt.Errorf("config file missing 'version' field (expected: %s)", ConfigVersion)
	}

	switch version {
	case "v1":
		return nil
	default:
		return fmt.Errorf("unsupported config version %q (supported: %s)", version, ConfigVersion)
	}
}

// ResolvePaths expands ~ in paths and sets defaults based on PKI directory.
func (cli *CLI) ResolvePaths() error {
	// Expand ~ in PKI directory
	if cli.PKI.Dir != "" {
		dir, err := expandHome(cli.PKI.Dir)
		if err != nil {
			return err
		}
		cli.PKI.Dir = dir
	}

	// Set defaults for PKI paths if not explicitly set
	if cli.PKI.CACert == "" {
		cli.PKI.CACert = filepath.Join(cli.PKI.Dir, "ca.crt")
	}
	if cli.PKI.CAKey == "" {
		cli.PKI.CAKey = filepath.Join(cli.PKI.Dir, "ca.key")
	}
	if cli.PKI.Cert == "" {
		cli.PKI.Cert = filepath.Join(cli.PKI.Dir, "latis.crt")
	}
	if cli.PKI.Key == "" {
		cli.PKI.Key = filepath.Join(cli.PKI.Dir, "latis.key")
	}

	return nil
}

// ResolvePeer returns the address for a peer (by name or direct address).
func (cli *CLI) ResolvePeer(nameOrAddr string) string {
	for _, p := range cli.Peers {
		if p.Name == nameOrAddr {
			return p.Addr
		}
	}
	return nameOrAddr
}

// Identity returns the node identity string, generating one if not set.
func (cli *CLI) Identity() string {
	name := cli.Agent.Name
	if name == "" {
		name = uuid.New().String()[:8]
	}
	return pki.UnitIdentity(name)
}

// CreateLLMProvider creates the configured LLM provider.
func (cli *CLI) CreateLLMProvider() (llm.Provider, error) {
	switch cli.LLM.Provider {
	case "":
		return nil, fmt.Errorf("--llm-provider is required (options: echo, ollama)")
	case "ollama":
		if cli.LLM.Model == "" {
			return nil, fmt.Errorf("--llm-model is required when using ollama provider")
		}
		url := cli.LLM.URL
		if url == "" {
			url = "http://localhost:11434/v1"
		}
		return llm.NewOllamaProvider(llm.OllamaConfig{
			BaseURL: url,
			Model:   cli.LLM.Model,
		}), nil
	case "echo":
		return llm.NewEchoProvider(), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cli.LLM.Provider)
	}
}

// AgentCard builds an A2A AgentCard from the configuration.
func (cli *CLI) AgentCard(addr string) *a2a.AgentCard {
	name := cli.Agent.Name
	if name == "" {
		name = "latis"
	}

	description := cli.Agent.Description
	if description == "" {
		description = fmt.Sprintf("Latis Agent (LLM: %s)", cli.LLM.Provider)
	}

	inputModes := cli.Agent.InputModes
	if len(inputModes) == 0 {
		inputModes = []string{"text"}
	}

	outputModes := cli.Agent.OutputModes
	if len(outputModes) == 0 {
		outputModes = []string{"text"}
	}

	card := &a2a.AgentCard{
		Name:               name,
		Description:        description,
		URL:                addr,
		PreferredTransport: a2a.TransportProtocolGRPC,
		DefaultInputModes:  inputModes,
		DefaultOutputModes: outputModes,
		Capabilities: a2a.AgentCapabilities{
			Streaming: cli.IsStreaming(),
		},
	}

	// Add skills from configuration
	for _, skill := range cli.Agent.Skills {
		card.Skills = append(card.Skills, a2a.AgentSkill{
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			Tags:        skill.Tags,
			Examples:    skill.Examples,
		})
	}

	return card
}

// expandHome expands ~ to the user's home directory.
func expandHome(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	return filepath.Join(home, path[1:]), nil
}
