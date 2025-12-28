// Package llm provides a pluggable interface for LLM providers.
package llm

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/mark3labs/mcphost/sdk"
	"gopkg.in/yaml.v3"
)

// MCPServerConfig represents configuration for an MCP server.
// Mirrors the mcphost config format for compatibility.
type MCPServerConfig struct {
	Type        string            `yaml:"type"`                  // "local", "remote", "builtin"
	Command     []string          `yaml:"command,omitempty"`     // for local
	Environment map[string]string `yaml:"environment,omitempty"` // for local
	URL         string            `yaml:"url,omitempty"`         // for remote
	Headers     []string          `yaml:"headers,omitempty"`     // for remote
	Name        string            `yaml:"name,omitempty"`        // for builtin
	Options     map[string]any    `yaml:"options,omitempty"`     // for builtin
}

// MCPHostOptions configures the MCPHostProvider.
type MCPHostOptions struct {
	// Model is the model string in "provider:model" format (e.g., "ollama:llama3.2")
	Model string

	// SystemPrompt is the optional system prompt for the agent
	SystemPrompt string

	// MCPConfigFile is the path to an external mcphost config file.
	// If set, this takes precedence over MCPServers.
	// This allows reusing a standalone mcphost config with Tndrl.
	MCPConfigFile string

	// MCPServers maps server names to their configurations.
	// Only used if MCPConfigFile is not set.
	MCPServers map[string]MCPServerConfig

	// MaxSteps limits the number of tool calls (0 for unlimited)
	MaxSteps int

	// Streaming enables streaming responses
	Streaming bool
}

// mcpHostConfig is the config file format expected by mcphost SDK
type mcpHostConfig struct {
	MCPServers map[string]MCPServerConfig `yaml:"mcpServers"`
}

// MCPHostProvider implements llm.Provider using the mcphost SDK.
// It wraps an MCPHost instance and handles the tool calling loop internally.
type MCPHostProvider struct {
	host       *sdk.MCPHost
	model      string
	configFile string // temp config file to clean up
	mu         sync.Mutex
}

// NewMCPHostProvider creates a new MCPHostProvider.
// If MCPConfigFile is set, it uses that external config file directly.
// Otherwise, it writes a temporary config file from MCPServers.
func NewMCPHostProvider(ctx context.Context, opts MCPHostOptions) (*MCPHostProvider, error) {
	if opts.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	var configFile string
	var tempFile string // only set if we created a temp file

	if opts.MCPConfigFile != "" {
		// Use external config file directly
		configFile = opts.MCPConfigFile
		slog.Debug("creating mcphost provider with external config",
			"model", opts.Model,
			"config_file", configFile,
		)
	} else {
		// Write MCP server config to a temp file
		var err error
		tempFile, err = writeTempConfig(opts.MCPServers)
		if err != nil {
			return nil, fmt.Errorf("failed to write temp config: %w", err)
		}
		configFile = tempFile
		slog.Debug("creating mcphost provider with embedded config",
			"model", opts.Model,
			"config_file", configFile,
			"server_count", len(opts.MCPServers),
		)
	}

	// Create mcphost SDK instance
	host, err := sdk.New(ctx, &sdk.Options{
		Model:        opts.Model,
		SystemPrompt: opts.SystemPrompt,
		ConfigFile:   configFile,
		MaxSteps:     opts.MaxSteps,
		Streaming:    opts.Streaming,
		Quiet:        true, // suppress debug output
	})
	if err != nil {
		// Clean up temp file on error (only if we created it)
		if tempFile != "" {
			os.Remove(tempFile)
		}
		return nil, fmt.Errorf("failed to create mcphost: %w", err)
	}

	return &MCPHostProvider{
		host:       host,
		model:      opts.Model,
		configFile: tempFile, // only clean up if we created it
	}, nil
}

// writeTempConfig writes MCP server configuration to a temporary YAML file.
func writeTempConfig(servers map[string]MCPServerConfig) (string, error) {
	cfg := mcpHostConfig{
		MCPServers: servers,
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	f, err := os.CreateTemp("", "tndrl-mcphost-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("failed to write config: %w", err)
	}

	return f.Name(), nil
}

// Complete generates a response for the given messages (non-streaming).
// The mcphost SDK handles the tool calling loop internally.
func (p *MCPHostProvider) Complete(ctx context.Context, messages []Message) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear session for fresh conversation
	p.host.ClearSession()

	// Build the conversation from messages
	// For now, we only use the last user message
	// TODO: Consider building full conversation history
	var userMessage string
	for _, msg := range messages {
		if msg.Role == "user" {
			userMessage = msg.Content
		}
	}

	if userMessage == "" {
		return "", fmt.Errorf("no user message found")
	}

	slog.Debug("mcphost complete", "message_length", len(userMessage))

	response, err := p.host.Prompt(ctx, userMessage)
	if err != nil {
		slog.Error("mcphost prompt failed", "err", err)
		return "", err
	}

	slog.Debug("mcphost response", "response_length", len(response))
	return response, nil
}

// Stream generates a streaming response.
// Uses mcphost's callback-based streaming.
func (p *MCPHostProvider) Stream(ctx context.Context, messages []Message) (<-chan StreamEvent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear session for fresh conversation
	p.host.ClearSession()

	// Build the conversation from messages
	var userMessage string
	for _, msg := range messages {
		if msg.Role == "user" {
			userMessage = msg.Content
		}
	}

	if userMessage == "" {
		return nil, fmt.Errorf("no user message found")
	}

	slog.Debug("mcphost stream", "message_length", len(userMessage))

	ch := make(chan StreamEvent, 16)

	go func() {
		defer close(ch)

		response, err := p.host.PromptWithCallbacks(ctx, userMessage,
			func(name, args string) {
				// Tool call started
				slog.Debug("mcphost tool call", "tool", name)
			},
			func(name, args, result string, isError bool) {
				// Tool result received
				slog.Debug("mcphost tool result", "tool", name, "is_error", isError)
			},
			func(chunk string) {
				// Streaming chunk received
				select {
				case ch <- StreamEvent{Content: chunk}:
				case <-ctx.Done():
					return
				}
			},
		)

		if err != nil {
			slog.Error("mcphost stream failed", "err", err)
			select {
			case ch <- StreamEvent{Error: err}:
			case <-ctx.Done():
			}
			return
		}

		// Send final event with done flag
		select {
		case ch <- StreamEvent{Content: response, Done: true}:
		case <-ctx.Done():
		}
	}()

	return ch, nil
}

// Name returns the provider identifier.
func (p *MCPHostProvider) Name() string {
	return "mcphost"
}

// Close cleans up resources including the temporary config file.
func (p *MCPHostProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error

	if p.host != nil {
		if err := p.host.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close mcphost: %w", err))
		}
	}

	if p.configFile != "" {
		if err := os.Remove(p.configFile); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("failed to remove temp config: %w", err))
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

var _ Provider = (*MCPHostProvider)(nil)
