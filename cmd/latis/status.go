package main

import (
	"context"
	"fmt"
	"log"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
)

// StatusCmd gets status from a peer.
type StatusCmd struct {
	Peer string `arg:"" help:"Peer address or name"`
}

// Run executes the status command.
func (c *StatusCmd) Run(cli *CLI) error {
	addr := cli.ResolvePeer(c.Peer)
	log.Printf("getting status from %s", addr)

	conn, err := ConnectToPeer(cli, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	return doGetStatus(context.Background(), conn.ControlClient())
}

func doGetStatus(ctx context.Context, client latisv1.ControlServiceClient) error {
	resp, err := client.GetStatus(ctx, &latisv1.GetStatusRequest{})
	if err != nil {
		return fmt.Errorf("get status failed: %w", err)
	}

	fmt.Printf("Status:\n")
	fmt.Printf("  Identity:     %s\n", resp.Identity)
	fmt.Printf("  State:        %s\n", resp.State.String())
	fmt.Printf("  Uptime:       %ds\n", resp.UptimeSeconds)
	fmt.Printf("  Active Tasks: %d\n", resp.ActiveTasks)
	if len(resp.Metadata) > 0 {
		fmt.Printf("  Metadata:\n")
		for k, v := range resp.Metadata {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
	return nil
}
