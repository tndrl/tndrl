package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/a2aproject/a2a-go/a2aclient"
)

// DiscoverCmd discovers a peer's capabilities by fetching its AgentCard.
type DiscoverCmd struct {
	Peer string `arg:"" help:"Peer address or name"`
}

// Run executes the discover command.
func (c *DiscoverCmd) Run(cli *CLI) error {
	addr := cli.ResolvePeer(c.Peer)
	log.Printf("discovering %s", addr)

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

	return doDiscover(context.Background(), transport)
}

func doDiscover(ctx context.Context, transport a2aclient.Transport) error {
	card, err := transport.GetAgentCard(ctx)
	if err != nil {
		return fmt.Errorf("get agent card failed: %w", err)
	}

	fmt.Printf("Agent: %s\n", card.Name)
	if card.Description != "" {
		fmt.Printf("Description: %s\n", card.Description)
	}
	if card.URL != "" {
		fmt.Printf("URL: %s\n", card.URL)
	}
	fmt.Printf("Transport: %s\n", card.PreferredTransport)

	if len(card.DefaultInputModes) > 0 {
		fmt.Printf("Input Modes: %s\n", strings.Join(card.DefaultInputModes, ", "))
	}
	if len(card.DefaultOutputModes) > 0 {
		fmt.Printf("Output Modes: %s\n", strings.Join(card.DefaultOutputModes, ", "))
	}

	fmt.Printf("Streaming: %v\n", card.Capabilities.Streaming)

	if len(card.Skills) > 0 {
		fmt.Printf("\nSkills:\n")
		for _, skill := range card.Skills {
			fmt.Printf("  - %s: %s\n", skill.ID, skill.Description)
			if len(skill.Tags) > 0 {
				fmt.Printf("    Tags: [%s]\n", strings.Join(skill.Tags, ", "))
			}
			if len(skill.Examples) > 0 {
				fmt.Printf("    Examples:\n")
				for _, ex := range skill.Examples {
					fmt.Printf("      - %s\n", ex)
				}
			}
		}
	}

	return nil
}
