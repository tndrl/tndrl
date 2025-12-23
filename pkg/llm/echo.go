package llm

import "context"

// EchoProvider is a simple provider that echoes input for testing.
type EchoProvider struct{}

// NewEchoProvider creates a new echo provider.
func NewEchoProvider() *EchoProvider {
	return &EchoProvider{}
}

// Complete echoes the last user message.
func (p *EchoProvider) Complete(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	// Find the last user message
	var content string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			content = messages[i].Content
			break
		}
	}

	return content, nil
}

// Stream echoes the last user message as a single event.
func (p *EchoProvider) Stream(ctx context.Context, messages []Message) (<-chan StreamEvent, error) {
	ch := make(chan StreamEvent, 1)

	go func() {
		defer close(ch)

		response, err := p.Complete(ctx, messages)
		if err != nil {
			ch <- StreamEvent{Error: err, Done: true}
			return
		}

		ch <- StreamEvent{Content: response, Done: true}
	}()

	return ch, nil
}

// Name returns the provider identifier.
func (p *EchoProvider) Name() string {
	return "echo"
}
