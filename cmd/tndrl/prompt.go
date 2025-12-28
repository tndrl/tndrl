package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
)

// PromptCmd sends a prompt to a peer via A2A.
type PromptCmd struct {
	Peer    string `arg:"" help:"Peer address or name"`
	Message string `arg:"" help:"Message to send"`
	Stream  bool   `help:"Use streaming response" short:"s"`
}

// Run executes the prompt command.
func (c *PromptCmd) Run(cli *CLI) error {
	addr := cli.ResolvePeer(c.Peer)
	slog.Debug("sending prompt", "addr", addr, "streaming", c.Stream)

	conn, err := ConnectToPeer(cli, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	transport, err := conn.A2ATransport()
	if err != nil {
		return err
	}
	defer transport.Destroy()

	ctx := context.Background()
	if c.Stream {
		return doStreamingPrompt(ctx, transport, c.Message)
	}
	return doPrompt(ctx, transport, c.Message)
}

func doPrompt(ctx context.Context, transport a2aclient.Transport, content string) error {
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: content})
	resp, err := transport.SendMessage(ctx, &a2a.MessageSendParams{Message: msg})
	if err != nil {
		return fmt.Errorf("send message failed: %w", err)
	}

	switch r := resp.(type) {
	case *a2a.Task:
		printTask(r)
	case *a2a.Message:
		printMessage(r)
	default:
		fmt.Printf("Response: %+v\n", resp)
	}
	return nil
}

func doStreamingPrompt(ctx context.Context, transport a2aclient.Transport, content string) error {
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: content})
	events := transport.SendStreamingMessage(ctx, &a2a.MessageSendParams{Message: msg})

	for event, err := range events {
		if err != nil {
			return fmt.Errorf("streaming error: %w", err)
		}

		switch e := event.(type) {
		case *a2a.TaskStatusUpdateEvent:
			if e.Status.Message != nil {
				for _, part := range e.Status.Message.Parts {
					if text, ok := part.(a2a.TextPart); ok {
						fmt.Print(text.Text)
					}
				}
			}
		case *a2a.TaskArtifactUpdateEvent:
			fmt.Printf("\n[artifact] %s\n", e.Artifact.Name)
		}
	}
	fmt.Println()
	return nil
}

func printTask(task *a2a.Task) {
	fmt.Printf("Task:\n")
	fmt.Printf("  ID:    %s\n", task.ID)
	fmt.Printf("  State: %s\n", task.Status.State)
	if task.Status.Message != nil {
		printMessage(task.Status.Message)
	}
}

func printMessage(msg *a2a.Message) {
	fmt.Printf("Message (role=%s):\n", msg.Role)
	for _, part := range msg.Parts {
		switch p := part.(type) {
		case a2a.TextPart:
			fmt.Printf("  %s\n", p.Text)
		default:
			fmt.Printf("  [%T]\n", part)
		}
	}
}
