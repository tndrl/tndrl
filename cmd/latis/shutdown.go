package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
)

// ShutdownCmd requests a peer to shutdown.
type ShutdownCmd struct {
	Peer     string `arg:"" help:"Peer address or name"`
	Force    bool   `help:"Force immediate shutdown (not graceful)"`
	Timeout  int    `help:"Graceful shutdown timeout in seconds" default:"30"`
	Reason   string `help:"Reason for shutdown" default:"requested by peer"`
}

// Run executes the shutdown command.
func (c *ShutdownCmd) Run(cli *CLI) error {
	addr := cli.ResolvePeer(c.Peer)
	slog.Debug("requesting shutdown", "addr", addr, "force", c.Force)

	conn, err := ConnectToPeer(cli, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	return doShutdown(context.Background(), conn.ControlClient(), !c.Force, int64(c.Timeout), c.Reason)
}

func doShutdown(ctx context.Context, client latisv1.ControlServiceClient, graceful bool, timeout int64, reason string) error {
	resp, err := client.Shutdown(ctx, &latisv1.ShutdownRequest{
		Graceful:       graceful,
		TimeoutSeconds: timeout,
		Reason:         reason,
	})
	if err != nil {
		return fmt.Errorf("shutdown request failed: %w", err)
	}

	if resp.Accepted {
		fmt.Println("shutdown accepted")
		return nil
	}

	fmt.Printf("shutdown rejected: %s\n", resp.RejectionReason)
	os.Exit(1)
	return nil
}
