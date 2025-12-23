package a2aexec

import (
	"context"
	"testing"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"

	"github.com/shanemcd/latis/pkg/llm"
)

// testQueue is a simple queue for testing that collects events.
type testQueue struct {
	events []a2a.Event
}

func (q *testQueue) Write(ctx context.Context, event a2a.Event) error {
	q.events = append(q.events, event)
	return nil
}

func (q *testQueue) Read(ctx context.Context) (a2a.Event, error) {
	if len(q.events) == 0 {
		return nil, context.Canceled
	}
	event := q.events[0]
	q.events = q.events[1:]
	return event, nil
}

func (q *testQueue) Close() error {
	return nil
}

func TestExecutor_Execute(t *testing.T) {
	exec := NewExecutor()

	msg := &a2a.Message{
		Role: a2a.MessageRoleUser,
		Parts: []a2a.Part{
			a2a.TextPart{Text: "Hello, world!"},
		},
	}

	reqCtx := &a2asrv.RequestContext{
		Message:   msg,
		TaskID:    "test-task-1",
		ContextID: "test-context-1",
	}

	// Create a queue to capture events
	q := &testQueue{}

	err := exec.Execute(context.Background(), reqCtx, q)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Read the response from the queue
	event, err := q.Read(context.Background())
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	response, ok := event.(*a2a.Message)
	if !ok {
		t.Fatalf("expected *a2a.Message, got %T", event)
	}

	if response.Role != a2a.MessageRoleAgent {
		t.Errorf("expected role %q, got %q", a2a.MessageRoleAgent, response.Role)
	}

	if len(response.Parts) == 0 {
		t.Fatal("expected at least one part in response")
	}

	text, ok := response.Parts[0].(a2a.TextPart)
	if !ok {
		t.Fatalf("expected TextPart, got %T", response.Parts[0])
	}

	expected := "Hello, world!"
	if text.Text != expected {
		t.Errorf("expected %q, got %q", expected, text.Text)
	}
}

func TestExecutor_Cancel(t *testing.T) {
	exec := NewExecutor()

	reqCtx := &a2asrv.RequestContext{
		TaskID:    "test-task-1",
		ContextID: "test-context-1",
	}

	q := &testQueue{}

	err := exec.Cancel(context.Background(), reqCtx, q)
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	event, err := q.Read(context.Background())
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	statusEvent, ok := event.(*a2a.TaskStatusUpdateEvent)
	if !ok {
		t.Fatalf("expected *a2a.TaskStatusUpdateEvent, got %T", event)
	}

	if statusEvent.Status.State != a2a.TaskStateCanceled {
		t.Errorf("expected state %q, got %q", a2a.TaskStateCanceled, statusEvent.Status.State)
	}

	if !statusEvent.Final {
		t.Error("expected Final to be true")
	}
}

// customProvider is a test provider that returns a fixed response.
type customProvider struct {
	response string
}

func (p *customProvider) Complete(ctx context.Context, messages []llm.Message) (string, error) {
	return p.response, nil
}

func (p *customProvider) Stream(ctx context.Context, messages []llm.Message) (<-chan llm.StreamEvent, error) {
	ch := make(chan llm.StreamEvent, 1)
	go func() {
		defer close(ch)
		ch <- llm.StreamEvent{Content: p.response, Done: true}
	}()
	return ch, nil
}

func (p *customProvider) Name() string { return "custom" }

func TestExecutor_CustomProvider(t *testing.T) {
	exec := &Executor{
		Provider: &customProvider{response: "Custom response"},
	}

	msg := &a2a.Message{
		Role: a2a.MessageRoleUser,
		Parts: []a2a.Part{
			a2a.TextPart{Text: "Test"},
		},
	}

	reqCtx := &a2asrv.RequestContext{
		Message:   msg,
		TaskID:    "test-task-1",
		ContextID: "test-context-1",
	}

	q := &testQueue{}

	err := exec.Execute(context.Background(), reqCtx, q)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	event, err := q.Read(context.Background())
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	response, ok := event.(*a2a.Message)
	if !ok {
		t.Fatalf("expected *a2a.Message, got %T", event)
	}

	text := response.Parts[0].(a2a.TextPart)
	if text.Text != "Custom response" {
		t.Errorf("expected %q, got %q", "Custom response", text.Text)
	}
}
