package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/shanemcd/latis/pkg/llm"
	"github.com/shanemcd/latis/pkg/pki"
)

// ConfigVersion is the current config file version.
const ConfigVersion = "v1"

// Config is the unified configuration for latis-unit.
// It serves as the single source of truth for CLI flags, env vars, and config files.
type Config struct {
	ConfigFile string `help:"Path to config file" short:"c" type:"path" yaml:"-"`

	// Version is the config file schema version.
	// Required in config files to ensure compatibility.
	Version string `yaml:"version" json:"version"`

	Addr   string `help:"Address to listen on" default:"[::]:4433" env:"LATIS_ADDR" yaml:"addr"`
	UnitID string `help:"Unit ID for certificate identity" env:"LATIS_UNIT_ID" yaml:"unitID"`

	PKI PKIConfig `embed:"" prefix:"pki-" yaml:"pki"`
	LLM LLMConfig `embed:"" prefix:"llm-" yaml:"llm"`

	Agent AgentConfig `embed:"" prefix:"agent-" yaml:"agent"`
}

// PKIConfig holds PKI-related configuration.
type PKIConfig struct {
	Dir    string `help:"PKI directory" default:"~/.latis/pki" env:"LATIS_PKI_DIR" yaml:"dir"`
	CACert string `help:"CA certificate path (overrides pki-dir)" env:"LATIS_CA_CERT" yaml:"caCert"`
	CAKey  string `help:"CA private key path" env:"LATIS_CA_KEY" yaml:"caKey"`
	Cert   string `help:"Unit certificate path" env:"LATIS_CERT" yaml:"cert"`
	Key    string `help:"Unit private key path" env:"LATIS_KEY" yaml:"key"`
	Init   bool   `help:"Initialize PKI if missing" env:"LATIS_INIT_PKI" yaml:"init"`
}

// LLMConfig holds LLM provider configuration.
type LLMConfig struct {
	Provider string `help:"LLM provider (echo, ollama)" default:"echo" env:"LATIS_LLM_PROVIDER" yaml:"provider"`
	Model    string `help:"LLM model name" env:"LATIS_LLM_MODEL" yaml:"model"`
	URL      string `help:"LLM API URL" default:"http://localhost:11434/v1" env:"LATIS_LLM_URL" yaml:"url"`
}

// AgentConfig holds A2A agent card configuration.
type AgentConfig struct {
	Name        string   `help:"Agent name" env:"LATIS_AGENT_NAME" yaml:"name"`
	Description string   `help:"Agent description" env:"LATIS_AGENT_DESCRIPTION" yaml:"description"`
	InputModes  []string `help:"Supported input modes" default:"text" env:"LATIS_AGENT_INPUT_MODES" yaml:"inputModes"`
	OutputModes []string `help:"Supported output modes" default:"text" env:"LATIS_AGENT_OUTPUT_MODES" yaml:"outputModes"`
	Streaming   bool     `help:"Enable streaming" default:"true" env:"LATIS_AGENT_STREAMING" yaml:"streaming"`
	Skills      []Skill  `yaml:"skills"`
}

// Skill represents an A2A agent skill.
type Skill struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Examples    []string `yaml:"examples"`
}

// LoadConfigFile loads configuration from a YAML file into the config struct.
// If the path is empty, this is a no-op.
// Returns an error if the config file version is not supported.
func LoadConfigFile(path string, cfg *Config) error {
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	// Validate version
	if err := validateConfigVersion(cfg.Version); err != nil {
		return err
	}

	return nil
}

// validateConfigVersion checks that the config file version is supported.
func validateConfigVersion(version string) error {
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
func (c *Config) ResolvePaths() error {
	// Expand ~ in PKI directory
	if c.PKI.Dir != "" {
		dir, err := expandHome(c.PKI.Dir)
		if err != nil {
			return err
		}
		c.PKI.Dir = dir
	}

	// Set defaults for PKI paths if not explicitly set
	if c.PKI.CACert == "" {
		c.PKI.CACert = filepath.Join(c.PKI.Dir, "ca.crt")
	}
	if c.PKI.CAKey == "" {
		c.PKI.CAKey = filepath.Join(c.PKI.Dir, "ca.key")
	}
	if c.PKI.Cert == "" {
		c.PKI.Cert = filepath.Join(c.PKI.Dir, "unit.crt")
	}
	if c.PKI.Key == "" {
		c.PKI.Key = filepath.Join(c.PKI.Dir, "unit.key")
	}

	return nil
}

// Identity returns the unit identity string, generating one if not set.
func (c *Config) Identity() string {
	id := c.UnitID
	if id == "" {
		id = uuid.New().String()[:8]
	}
	return pki.UnitIdentity(id)
}

// CreateLLMProvider creates the configured LLM provider.
func (c *Config) CreateLLMProvider() (llm.Provider, error) {
	switch c.LLM.Provider {
	case "ollama":
		if c.LLM.Model == "" {
			return nil, fmt.Errorf("--llm-model is required when using ollama provider")
		}
		return llm.NewOllamaProvider(llm.OllamaConfig{
			BaseURL: c.LLM.URL,
			Model:   c.LLM.Model,
		}), nil
	case "echo":
		return llm.NewEchoProvider(), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", c.LLM.Provider)
	}
}

// AgentCard builds an A2A AgentCard from the configuration.
func (c *Config) AgentCard(addr string) *a2a.AgentCard {
	name := c.Agent.Name
	if name == "" {
		name = "latis-unit"
	}

	description := c.Agent.Description
	if description == "" {
		description = fmt.Sprintf("Latis Unit Agent (LLM: %s)", c.LLM.Provider)
	}

	inputModes := c.Agent.InputModes
	if len(inputModes) == 0 {
		inputModes = []string{"text"}
	}

	outputModes := c.Agent.OutputModes
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
			Streaming: c.Agent.Streaming,
		},
	}

	// Add skills from configuration
	for _, skill := range c.Agent.Skills {
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
