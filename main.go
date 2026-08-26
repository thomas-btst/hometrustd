package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	godbus "github.com/godbus/dbus/v5"
)

const ProgramName = "hometrustd"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	conn, err := godbus.ConnectSystemBus()
	if err != nil {
		slog.Error("Failed to connect to system dbus", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("Failed to cleanly close dbus connection", slog.Any("error", err))
		}
	}()

	networkMonitor := NewNetworkMonitor(conn)
	wifiStates, err := networkMonitor.Watch(ctx)
	if err != nil {
		slog.Error("Failed to start network monitor watch", slog.Any("error", err))
		os.Exit(1)
	}

	for state := range wifiStates {
		fmt.Printf("Network state: %s\n", state)
	}
}
