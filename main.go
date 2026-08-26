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

	systemConn, err := godbus.ConnectSystemBus()
	if err != nil {
		slog.Error("Failed to connect to system dbus", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := systemConn.Close(); err != nil {
			slog.Error("Failed to cleanly close system dbus connection", slog.Any("error", err))
		}
	}()

	sessionConn, err := godbus.ConnectSessionBus()
	if err != nil {
		slog.Error("Failed to connect to session dbus", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := sessionConn.Close(); err != nil {
			slog.Error("Failed to cleanly close session dbus connection", slog.Any("error", err))
		}
	}()

	networkMonitor := NewNetworkMonitor(systemConn)

	wifiStates, err := networkMonitor.Watch(ctx)
	if err != nil {
		slog.Error("Failed to start network monitor watch", slog.Any("error", err))
		os.Exit(1)
	}

	idleInhibitor := NewIdleInhibitor(sessionConn)
	for state := range wifiStates {
		fmt.Printf("Network state: %s\n", state)

		if state.Connected {
			if err := idleInhibitor.Inhibit(
				fmt.Sprintf(
					"Connected to Wi-Fi network %s (%s)",
					state.SSID, state.BSSID,
				),
			); err != nil {
				slog.Error("Failed to inhibit idle", slog.Any("error", err))
			}
		} else {
			if err := idleInhibitor.Uninhibit(); err != nil {
				slog.Error("Failed to uninhibit idle", slog.Any("error", err))
			}
		}
	}
}
