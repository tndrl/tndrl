package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tndrlv1 "github.com/shanemcd/tndrl/gen/go/tndrl/v1"
)

// PingCmd sends a ping to a peer.
type PingCmd struct {
	Peer string `arg:"" help:"Peer address or name to ping"`
}

// Run executes the ping command.
func (c *PingCmd) Run(cli *CLI) error {
	addr := cli.ResolvePeer(c.Peer)
	slog.Debug("pinging peer", "addr", addr)

	conn, err := ConnectToPeer(cli, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	return doPing(context.Background(), conn.ControlClient())
}

func doPing(ctx context.Context, client tndrlv1.ControlServiceClient) error {
	pingTime := time.Now()
	resp, err := client.Ping(ctx, &tndrlv1.PingRequest{
		Timestamp: pingTime.UnixNano(),
	})
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	rtt := time.Since(pingTime)
	serverLatency := time.Duration(resp.PongTimestamp - resp.PingTimestamp)

	fmt.Printf("pong: rtt=%v, server_latency=%v\n", rtt, serverLatency)
	return nil
}
