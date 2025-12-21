// Package a2aexec implements the A2A AgentExecutor interface for Latis units.
package a2aexec

import (
	"context"
	"fmt"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
)

// Executor implements a2asrv.AgentExecutor for Latis units.
// It handles incoming A2A messages and produces responses.
type Executor struct {
	// Handler is called for each incoming message. If nil, a default echo handler is used.
	Handler func(ctx context.Context, msg *a2a.Message) (*a2a.Message, error)
}

// NewExecutor creates a new Executor with the default echo handler.
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute implements a2asrv.AgentExecutor.
// It processes the incoming message and writes response events to the queue.
func (e *Executor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue) error {
	msg := reqCtx.Message

	// Get the handler (use default echo if none set)
	handler := e.Handler
	if handler == nil {
		handler = defaultHandler
	}

	// Process the message
	response, err := handler(ctx, msg)
	if err != nil {
		// Report failure via status update
		failEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateFailed, &a2a.Message{
			Role: a2a.MessageRoleAgent,
			Parts: []a2a.Part{
				a2a.TextPart{Text: err.Error()},
			},
		})
		failEvent.Final = true
		return q.Write(ctx, failEvent)
	}

	// Write the response message
	return q.Write(ctx, response)
}

// Cancel implements a2asrv.AgentExecutor.
// For now, it simply acknowledges the cancellation request.
func (e *Executor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue) error {
	event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCanceled, nil)
	event.Final = true
	return q.Write(ctx, event)
}

// defaultHandler echoes the input message content.
func defaultHandler(ctx context.Context, msg *a2a.Message) (*a2a.Message, error) {
	// Extract text content from message parts
	var content string
	for _, part := range msg.Parts {
		if text, ok := part.(a2a.TextPart); ok {
			content = text.Text
			break
		}
	}

	// Echo the content back
	response := a2a.NewMessage(a2a.MessageRoleAgent, a2a.TextPart{
		Text: fmt.Sprintf("Echo: %s", content),
	})

	return response, nil
}
