// Package cmd implements the command line interface for hometrustd
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"
	"github.com/thomas-btst/hometrustd/internal/daemon"
	"github.com/thomas-btst/hometrustd/internal/idle"
	"github.com/thomas-btst/hometrustd/internal/network"
)

var rootCmd = &cobra.Command{
	Use:   "hometrustd",
	Short: "Daemon monitoring network status to manage system idle inhibition",
	Long: `HomeTrustd is a Linux daemon that monitors network connectivity (via NetworkManager)
and automatically manages system idle inhibition (via D-Bus) based on trusted Wi-Fi networks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		systemConn, err := dbus.ConnectSystemBus()
		if err != nil {
			return fmt.Errorf("failed to connect to system dbus: %w", err)
		}
		defer func() {
			if err := systemConn.Close(); err != nil {
				slog.Error("Failed to cleanly close system dbus connection", slog.Any("error", err))
			}
		}()

		sessionConn, err := dbus.ConnectSessionBus()
		if err != nil {
			return fmt.Errorf("failed to connect to session dbus: %w", err)
		}
		defer func() {
			if err := sessionConn.Close(); err != nil {
				slog.Error("Failed to cleanly close session dbus connection", slog.Any("error", err))
			}
		}()

		netMon := network.NewMonitor(systemConn)
		idleInh := idle.NewInhibitor(sessionConn)

		app := daemon.NewApp(netMon, idleInh)

		if err := app.Run(cmd.Context()); err != nil {
			return fmt.Errorf("failed to run daemon: %w", err)
		}

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
