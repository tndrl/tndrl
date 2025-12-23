// Package llm provides a pluggable interface for LLM providers.
package llm

import "context"

// Provider generates LLM completions.
type Provider interface {
	// Complete generates a response for the given messages (non-streaming).
	Complete(ctx context.Context, messages []Message) (string, error)

	// Stream generates a streaming response.
	// The returned channel receives events until the response is complete or an error occurs.
	// The channel is closed when streaming is done.
	Stream(ctx context.Context, messages []Message) (<-chan StreamEvent, error)

	// Name returns the provider identifier (e.g., "ollama", "openai").
	Name() string
}

// Message represents a conversation message.
type Message struct {
	Role    string // "user", "assistant", "system"
	Content string
}

// StreamEvent represents a chunk of streaming response.
type StreamEvent struct {
	Content string // text chunk (may be empty for final event)
	Done    bool   // true if this is the final event
	Error   error  // non-nil if an error occurred
}
