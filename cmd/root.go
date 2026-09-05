// Package cmd implements the command line interface for hometrustd
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thomas-btst/hometrustd/internal/config"
	"github.com/thomas-btst/hometrustd/internal/daemon"
	"github.com/thomas-btst/hometrustd/internal/idle"
	"github.com/thomas-btst/hometrustd/internal/network"
	"github.com/thomas-btst/hometrustd/internal/notify"
)

var rootCmd = &cobra.Command{
	Use:   "hometrustd",
	Short: "Daemon monitoring network status to manage system idle inhibition",
	Long: `HomeTrust Daemon is a Linux daemon that monitors network connectivity (via NetworkManager) and
automatically manages system idle inhibition (via D-Bus) based on trusted Wi-Fi networks.`,
	Example: `  # Start daemon with trusted Wi-Fi BSSIDs and optional aliases
  hometrustd -t 00:11:22:33:44:55=Home,66:77:88:99:AA:BB`,
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
		idleMon := idle.NewMonitor(sessionConn)
		idleInh := idle.NewInhibitor(sessionConn, idleMon)

		cfgStore, err := config.LoadAndWatch()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		notifSend := notify.NewClient(sessionConn, cfgStore)
		app := daemon.NewApp(netMon, idleInh, notifSend, cfgStore)

		if err := app.Run(cmd.Context()); err != nil {
			return fmt.Errorf("failed to run daemon: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.Flags().BoolP("quiet", "q", false, "Disable desktop notifications")
	if err := viper.BindPFlag("quiet", rootCmd.Flags().Lookup("quiet")); err != nil {
		slog.Error(
			"Failed to bind quiet flag to viper",
			slog.Any("error", err),
		)
	}

	rootCmd.Flags().StringToStringP(
		"trusted-bssids",
		"t",
		nil,
		"Comma-separated list of trusted Wi-Fi BSSIDs with optional aliases",
	)
	if err := viper.BindPFlag("trusted_networks.bssids", rootCmd.Flags().Lookup("trusted-bssids")); err != nil {
		slog.Error(
			"Failed to bind trusted-bssids flag to viper",
			slog.Any("error", err),
		)
	}
}

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
