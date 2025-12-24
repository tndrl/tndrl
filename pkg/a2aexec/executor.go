// Package a2aexec implements the A2A AgentExecutor interface for Latis nodes.
package a2aexec

import (
	"context"
	"strings"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"

	"github.com/shanemcd/latis/pkg/llm"
)

// Executor implements a2asrv.AgentExecutor for Latis nodes.
// It handles incoming A2A messages and produces responses using an LLM provider.
type Executor struct {
	// Provider is the LLM provider to use. If nil, an echo provider is used.
	Provider llm.Provider

	// Streaming enables streaming responses when true.
	Streaming bool
}

// NewExecutor creates a new Executor with the default echo provider.
func NewExecutor() *Executor {
	return &Executor{
		Provider: llm.NewEchoProvider(),
	}
}

// Execute implements a2asrv.AgentExecutor.
// It processes the incoming message and writes response events to the queue.
func (e *Executor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue) error {
	msg := reqCtx.Message

	// Extract text content from the message
	content := extractTextContent(msg)

	// Convert to LLM message format
	messages := []llm.Message{
		{Role: "user", Content: content},
	}

	provider := e.Provider
	if provider == nil {
		provider = llm.NewEchoProvider()
	}

	if e.Streaming {
		return e.executeStreaming(ctx, reqCtx, q, provider, messages)
	}

	return e.executeNonStreaming(ctx, reqCtx, q, provider, messages)
}

// executeNonStreaming handles non-streaming execution.
func (e *Executor) executeNonStreaming(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue, provider llm.Provider, messages []llm.Message) error {
	response, err := provider.Complete(ctx, messages)
	if err != nil {
		return e.writeError(ctx, reqCtx, q, err)
	}

	// Write the response message
	responseMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.TextPart{
		Text: response,
	})

	return q.Write(ctx, responseMsg)
}

// executeStreaming handles streaming execution.
func (e *Executor) executeStreaming(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue, provider llm.Provider, messages []llm.Message) error {
	stream, err := provider.Stream(ctx, messages)
	if err != nil {
		return e.writeError(ctx, reqCtx, q, err)
	}

	// Accumulate the full response for the final message
	var fullResponse strings.Builder

	for event := range stream {
		if event.Error != nil {
			return e.writeError(ctx, reqCtx, q, event.Error)
		}

		if event.Content != "" {
			fullResponse.WriteString(event.Content)

			// Send a status update with the current content
			statusEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateWorking, &a2a.Message{
				Role: a2a.MessageRoleAgent,
				Parts: []a2a.Part{
					a2a.TextPart{Text: event.Content},
				},
			})
			if err := q.Write(ctx, statusEvent); err != nil {
				return err
			}
		}

		if event.Done {
			// Send the final completed status
			finalEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCompleted, &a2a.Message{
				Role: a2a.MessageRoleAgent,
				Parts: []a2a.Part{
					a2a.TextPart{Text: fullResponse.String()},
				},
			})
			finalEvent.Final = true
			return q.Write(ctx, finalEvent)
		}
	}

	// Stream ended without explicit done
	finalEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCompleted, &a2a.Message{
		Role: a2a.MessageRoleAgent,
		Parts: []a2a.Part{
			a2a.TextPart{Text: fullResponse.String()},
		},
	})
	finalEvent.Final = true
	return q.Write(ctx, finalEvent)
}

// writeError writes an error status update to the queue.
func (e *Executor) writeError(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue, err error) error {
	failEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateFailed, &a2a.Message{
		Role: a2a.MessageRoleAgent,
		Parts: []a2a.Part{
			a2a.TextPart{Text: err.Error()},
		},
	})
	failEvent.Final = true
	return q.Write(ctx, failEvent)
}

// Cancel implements a2asrv.AgentExecutor.
// For now, it simply acknowledges the cancellation request.
func (e *Executor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue) error {
	event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCanceled, nil)
	event.Final = true
	return q.Write(ctx, event)
}

// extractTextContent extracts the text content from an A2A message.
func extractTextContent(msg *a2a.Message) string {
	var content string
	for _, part := range msg.Parts {
		if text, ok := part.(a2a.TextPart); ok {
			content = text.Text
			break
		}
	}
	return content
}
